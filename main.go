package main

import (
	"ankiSyncGo/internal/auth"
	"ankiSyncGo/internal/db"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/blockloop/scan"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// parse dotEnv if present
	_ = godotenv.Load()

	// handle db
	db.Initialize()

	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.String(200, "Anki Sync server written in Go 🚀")
	})
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.File("favicon.svg")
	})

	// ONLY FOR DEVELOPING!!
	if os.Getenv("BUILD") == "DEV" {
		r.POST("/register", func(c *gin.Context) {
			var postData struct {
				U string `json:"u"`
				P string `json:"p"`
			}
			if err := c.BindJSON(&postData); err != nil {
				_ = c.AbortWithError(400, errors.New("Missing parameters"))
				return
			}
			if len(postData.U) == 0 || len(postData.P) == 0 {
				_ = c.AbortWithError(400, errors.New("Missing parameters"))
				return
			}
			if err := auth.AddUser(postData.U, postData.P); err != nil {
				log.Fatal(err)
			}
		})
	}
	r.POST("/sync/hostKey", func(c *gin.Context) {

		var data struct {
			U string `json:"u"` // Username
			P string `json:"p"` // Password
		}

		if err := getData(c, &data); err != nil {
			_ = c.AbortWithError(400, errors.New("Unable to unmarshal"))
		}

		if data.U == "" {
			_ = c.AbortWithError(401, errors.New("Unauthorized"))
			return
		}

		if data.P == "" {
			_ = c.AbortWithError(401, errors.New("Unauthorized"))
			return
		}

		if !auth.ValidateUser(data.U, data.P) {
			_ = c.AbortWithError(401, errors.New("Wrong username or password"))
			return
		}

		_, err := db.DB.Exec("INSERT INTO sessions (username) VALUES ($1)", data.U)
		if err != nil {
			log.Println(err)
		}
		var skey string
		row := db.DB.QueryRow(`SELECT skey FROM sessions WHERE username=$1`, data.U)
		err = row.Scan(&skey)
		if err != nil {
			log.Fatal(err)
		}

		c.JSON(200, struct {
			Key string `json:"key"`
		}{skey})
	})

	r.POST("/sync/meta", func(c *gin.Context) {

		data, err := getSession(c)
		if err != nil {
			return
		}

		collection := struct {
			Cont    bool   `json:"cont"`
			HostNum int    `json:"hostNum"`
			Mod     int    `json:"mod"`
			Msg     string `json:"msg"`
			Scm     int64  `json:"scm"`
			Usn     int    `json:"usn"`
			Ts      int64  `json:"ts"`
			Uname   string `json:"uname"`
		}{
			true,
			1,
			0,
			"",
			time.Now().Unix(),
			0,
			time.Now().Unix(),
			data.Username,
		}

		// Get collection
		row := db.DB.QueryRow(`SELECT mod, scm, usn FROM col WHERE username=$1`, data.Username)
		if err := row.Scan(&collection.Mod, &collection.Scm, &collection.Usn); err != nil && err != sql.ErrNoRows {
			log.Fatal(err)
		}
		log.Printf("%+v", collection)
		c.JSON(200, collection)
	})

	r.POST("/sync/upload", func(c *gin.Context) {
		sesh, err := getSession(c)
		if err != nil {
			return
		}

		file, _, _ := c.Request.FormFile("data")
		var b []byte

		// decompress if compressed
		gr, _ := gzip.NewReader(file)
		b, _ = ioutil.ReadAll(gr)
		//log.Printf("%+v", string(b))

		// create temporary file to store the sqlite db
		f, err := ioutil.TempFile("", "anki-sync-go")
		if err != nil {
			// TODO: return http error code
			return
		}

		// write transmitted db to tempfile
		f.Write(b)

		sqlite, err := sql.Open("sqlite3", f.Name())
		if err != nil {
			// TODO: actual error handling???
			log.Println(err)
		}

		// TODO: check integrity of sqlite file

		// delete entries of user first
		_, err = db.DB.Exec(`DELETE FROM cards WHERE username=$1`, sesh.Username)
		if err != nil {
			// TODO: error handling
			log.Fatal(err)
		}
		rows, err := sqlite.Query(`SELECT * FROM cards;`)
		if err != nil {
			// TODO: error handling
			log.Fatal(err)
		}
		defer rows.Close()

		var cards []dbCard
		scan.Rows(&cards, rows)

		for _, card := range cards {
			// add card to user in our db
			log.Println(card)
			// TODO: for the love of god, refactor
			db.DB.Exec(`INSERT INTO cards ( username, id, nid, did, ord, mod,
					usn, type, queue, due, ivl, factor, reps, lapses,
					"left", odue, odid, flags, data
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19);`,
				sesh.Username, card.Id, card.Nid, card.Did, card.Ord,
				card.Mod, card.Usn, card.Type, card.Queue, card.Due, card.Ivl,
				card.Factor, card.Reps, card.Lapses, card.Left, card.Odue, card.Odid, card.Flags, card.Data)
		}

		// TODO: the other tables, such as notes, col etc

		///c.JSON(200, collection)
		c.String(200, "OK")
	})

	r.POST("/msync/begin", func(c *gin.Context) {
		
	})

	srv := &http.Server{
		Addr:    ":27701",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	<-c
	log.Println("Shutting down...")
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func getSession(c *gin.Context) (session, error) {

	providedKey := c.Request.FormValue("k")

	var data session

	row := db.DB.QueryRow(`SELECT * FROM sessions WHERE skey=$1`, providedKey)
	err := row.Scan(&data.Skey, &data.Username, &data.created)
	if err != nil {
		log.Fatal(err)
		return session{}, err
	}
	return data, nil
}

func getData(c *gin.Context, target interface{}) error {

	file, _, _ := c.Request.FormFile("data")
	var b []byte

	// decompress if compressed
	gr, _ := gzip.NewReader(file)
	b, _ = ioutil.ReadAll(gr)

	if err := json.Unmarshal(b, target); err != nil {
		return err
	}
	return nil
}
