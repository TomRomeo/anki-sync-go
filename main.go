package main

import (
	"ankiSyncGo/internal/auth"
	"ankiSyncGo/internal/db"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	uuid "github.com/satori/go.uuid"
	sqlite2 "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
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

		var data db.Auth

		rawData := getData(c)
		json.Unmarshal(rawData, &data)

		if data.Username == "" {
			_ = c.AbortWithError(401, errors.New("Unauthorized"))
			return
		}

		if data.Pw == "" {
			_ = c.AbortWithError(401, errors.New("Unauthorized"))
			return
		}

		if !auth.ValidateUser(data.Username, data.Pw) {
			_ = c.AbortWithError(401, errors.New("Wrong username or password"))
			return
		}

		sesh := db.Session{Username: data.Username}
		db.DB.Create(&sesh)

		db.DB.First(&sesh, "username = ?", data.Username)

		log.Println(sesh)

		c.JSON(200, struct {
			Key uuid.UUID `json:"key"`
		}{sesh.Skey})
	})

	r.POST("/sync/meta", func(c *gin.Context) {

		sesh, ok := getSession(c)
		if !ok {
			log.Fatal("Could not retrieve session")
		}

		// Get collection
		var col db.Col
		db.DB.Where(&db.Col{Username: sesh.Username})

		resp := struct {
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
			col.Mod,
			"",
			time.Now().Unix(),
			col.Usn,
			time.Now().Unix(),
			sesh.Username,
		}

		log.Printf("%+v", resp)
		c.JSON(200, resp)
	})

	r.POST("/sync/upload", func(c *gin.Context) {
		sesh, ok := getSession(c)
		if !ok {
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
		defer os.Remove(f.Name())

		// write transmitted db to tempfile
		f.Write(b)

		sqlite, err := gorm.Open(sqlite2.Open(f.Name()))
		if err != nil {
			// TODO: actual error handling???
			log.Println(err)
		}

		// TODO: check integrity of sqlite file

		// delete entries of user first
		db.DB.Delete(&db.Col{}, "username = ?", sesh.Username)
		db.DB.Delete(&db.Card{}, "username = ?", sesh.Username)
		db.DB.Delete(&db.Note{}, "username = ?", sesh.Username)
		db.DB.Delete(&db.Revlog{}, "username = ?", sesh.Username)
		db.DB.Delete(&db.Media{}, "username = ?", sesh.Username)

		var cards []db.Card
		var col db.Col

		sqlite.Find(&cards)
		db.DB.Create(&cards)
		db.DB.Model(db.Card{}).Where("1 = 1").Updates(db.Card{Username: sesh.Username})

		sqlite.Find(&col)
		col.Username = sesh.Username
		db.DB.Create(&col)

		var notes []db.Note

		sqlite.Find(&notes)
		db.DB.Create(&notes)
		db.DB.Model(db.Note{}).Where("1 = 1").Updates(db.Note{Username: sesh.Username})

		var revlogs []db.Revlog
		sqlite.Find(&revlogs)

		db.DB.CreateInBatches(&revlogs, 1000)
		db.DB.Model(db.Revlog{}).Where("1 = 1").Updates(db.Revlog{Username: sesh.Username})

		c.String(200, "OK")
	})

	r.POST("/msync/begin", func(c *gin.Context) {
		sesh, ok := getSession(c)
		if !ok {
			// TODO: error handling
			log.Fatal("Could not find session")
		}

		var col db.Col
		db.DB.First(&col)

		c.JSON(200,
			struct {
				Data struct {
					Sk  uuid.UUID `json:"sk"`
					Usn int       `json:"usn"`
				} `json:"data"`
				Err string `json:"err"`
			}{
				struct {
					Sk  uuid.UUID `json:"sk"`
					Usn int       `json:"usn"`
				}{Sk: sesh.Skey, Usn: col.Usn},
				"",
			})
	})

	r.POST("/msync/mediaChanges", func(c *gin.Context) {
		_, ok := getSession(c)
		if !ok {
			// TODO: error handling
			log.Fatal("session not founddd")
		}
		var col db.Col
		db.DB.First(&col)

		t := struct {
			LastUsn int `json:"lastUsn"`
		}{}

		rawData := getData(c)
		json.Unmarshal(rawData, &t)
		// TODO: validate with own last media usn
		data := []string{}
		dat := struct {
			Data []string `json:"data"`
		}{
			Data: data,
		}
		c.JSON(200, dat)

	})
	r.POST("/msync/uploadChanges", func(c *gin.Context) {

		sesh, ok := getSession(c)
		if !ok {
			// TODO: error handling
			log.Fatal("session not found")
		}

		// TODO: refactor getData()
		file, _, _ := c.Request.FormFile("data")
		b, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatal(err)
		}
		usn := getLastMediaUsn()
		oldUsn := usn

		tmpZip, err := ioutil.TempFile("", "anki-sync-go-media")
		ioutil.WriteFile(tmpZip.Name(), b, 666)
		if err != nil {
			log.Fatal(err)
		}
		defer os.Remove(tmpZip.Name())
		zipReader, err := zip.OpenReader(tmpZip.Name())
		if err != nil {
			log.Fatal(err)
		}

		metaFile, err := zipReader.Open("_meta")
		if err != nil {
			log.Fatal(err)
		}
		metaFiles := [][]string{}
		ordToFilename := map[string]string{}
		metaFileContent, err := ioutil.ReadAll(metaFile)
		if err != nil {
			log.Fatal(err)
		}
		json.Unmarshal(metaFileContent, &metaFiles)

		filesToRemove := []string{}
		filesToAdd := []db.Media{}

		// if file does not have ordinal, delete that file
		for _, f := range metaFiles {
			if len(f) < 2 {
				filesToRemove = append(filesToRemove, f[0])
			}
			ordToFilename[f[1]] = f[0]
		}

		for _, f := range zipReader.File {
			if f.Name == "_meta" {
				continue
			}
			mediaFile, err := f.Open()
			if err != nil {
				log.Fatal(err)
			}
			mediaFileContent, err := ioutil.ReadAll(mediaFile)
			if err != nil {
				log.Fatal(err)
			}

			// write media file
			userDir := auth.GetUserDir(sesh.Username)
			ioutil.WriteFile(path.Join(userDir, ordToFilename[f.Name]), mediaFileContent, 666)
			usn++

			media := db.Media{
				Username: sesh.Username,
				Fname:    ordToFilename[f.Name],
				Usn:      usn,
				Csum:     checksum(mediaFileContent),
			}

			filesToAdd = append(filesToAdd, media)

			// write media to media db
			db.DB.Create(&media)
		}

		processedFiles := len(filesToRemove) + len(filesToAdd)

		if len(metaFiles) != processedFiles {
			log.Fatal(err)
		}

		// delete removed files
		for _, f := range filesToRemove {
			os.Remove(path.Join(auth.GetUserDir(sesh.Username), f))
		}

		if getLastMediaUsn() != oldUsn+processedFiles {
			log.Fatal("Wrong usn")
		}
		c.JSON(200, struct {
			Data  []int  `json:"data"`
			Error string `json:"error"`
		}{
			Data:  []int{processedFiles, usn},
			Error: "",
		})

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

func getSession(c *gin.Context) (db.Session, bool) {

	providedKey := c.Request.FormValue("k")

	if providedKey == "" {
		providedKey = c.Request.FormValue("sk")
	}

	var data db.Session
	var ok bool

	db.DB.First(&data, "skey = ?", providedKey)

	if data != (db.Session{}) {
		ok = true
	}
	return data, ok
}

func getData(c *gin.Context) []byte {

	file, _, _ := c.Request.FormFile("data")
	var b []byte

	// decompress if compressed
	gr, _ := gzip.NewReader(file)
	b, _ = ioutil.ReadAll(gr)

	return b
}

func checksum(data []byte) string {
	h := sha1.New()
	h.Write(data)
	sum := h.Sum(nil)
	return fmt.Sprintf("%x", sum)
}

func getLastMediaUsn() int {
	usn := 0
	rw := db.DB.Model(&db.Media{}).Where("1 = 1").Select("max(usn)").Row()
	rw.Scan(&usn)
	return usn
}
