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
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
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

	r.POST("/sync/meta", sessionMiddleware, func(c *gin.Context) {

		s, _ := c.Get("session")
		sesh, ok := s.(db.Session)
		if !ok {
			c.String(401, "Malformed Session")
			return
		}

		// Get collection
		var col db.Col
		db.DB.Find(&col, "username = ?", sesh.Username)

		resp := struct {
			Cont    bool   `json:"cont"`
			HostNum int    `json:"hostNum"`
			Mod     int    `json:"mod"`
			Msg     string `json:"msg"`
			Scm     int    `json:"scm"`
			Usn     int    `json:"usn"`
			Ts      int64  `json:"ts"`
			Uname   string `json:"uname"`
		}{
			true,
			1,
			col.Mod,
			"",
			col.Scm,
			col.Usn,
			time.Now().Unix(),
			sesh.Username,
		}

		log.Printf("%+v", resp)
		c.JSON(200, resp)
	})

	r.POST("/sync/upload", sessionMiddleware, func(c *gin.Context) {
		s, _ := c.Get("session")
		sesh, ok := s.(db.Session)
		if !ok {
			c.String(401, "Malformed Session")
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

	r.POST("/msync/begin", sessionMiddleware, func(c *gin.Context) {
		s, _ := c.Get("session")
		sesh, ok := s.(db.Session)
		if !ok {
			c.String(401, "Malformed Session")
			return
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

	r.POST("/msync/mediaChanges", sessionMiddleware, func(c *gin.Context) {
		s, _ := c.Get("session")
		sesh, ok := s.(db.Session)
		if !ok {
			c.String(401, "Malformed Session")
			return
		}

		var col db.Col
		db.DB.First(&col)

		t := struct {
			LastUsn int `json:"lastUsn"`
		}{}

		rawData := getData(c)
		json.Unmarshal(rawData, &t)
		// TODO: validate with own last media usn
		serverLastUsn := getLastMediaUsn()

		// get changes since last sync
		var missingMedia []db.SQLiteMedia
		outputFormattedMedia := [][]string{}
		if t.LastUsn < serverLastUsn || t.LastUsn == 0 {
			db.DB.Find(&missingMedia, "username = ? AND usn > ?", sesh.Username, t.LastUsn)
		}
		for _, media := range missingMedia {
			outputFormattedMedia = append(outputFormattedMedia, []string{media.Fname, strconv.FormatInt(int64(media.Usn), 10), media.Csum})
		}

		dat := struct {
			Data [][]string `json:"data"`
		}{
			Data: outputFormattedMedia,
		}
		c.JSON(200, dat)

	})
	r.POST("/msync/uploadChanges", sessionMiddleware, func(c *gin.Context) {
		s, _ := c.Get("session")
		sesh, ok := s.(db.Session)
		if !ok {
			c.String(401, "Malformed Session")
			return
		}

		// TODO: refactor getData()
		file, _, _ := c.Request.FormFile("data")
		b, err := ioutil.ReadAll(file)
		if err != nil {
			c.String(400, "Error parsing the submitted formfile")
			return
		}
		usn := getLastMediaUsn()
		oldUsn := usn

		tmpZip, err := ioutil.TempFile("", "anki-sync-go-media")
		ioutil.WriteFile(tmpZip.Name(), b, 666)
		if err != nil {
			c.String(500, "Could not create temporary zip file")
			return
		}
		defer os.Remove(tmpZip.Name())
		zipReader, err := zip.OpenReader(tmpZip.Name())
		if err != nil {
			c.String(500, "Could not create temporary zip file")
			return
		}

		metaFile, err := zipReader.Open("_meta")
		if err != nil {
			c.String(500, "Could not create temporary zip file")
			return
		}
		metaFiles := [][]string{}
		ordToFilename := map[string]string{}
		metaFileContent, err := ioutil.ReadAll(metaFile)
		if err != nil {
			c.String(500, "Could not create temporary zip file")
			return
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
				c.String(500, "Could not find media file")
				return
			}
			mediaFileContent, err := ioutil.ReadAll(mediaFile)
			if err != nil {
				c.String(500, "Could not read media file")
				return
			}

			// write media file
			userDir := auth.GetUserDir(sesh.Username)
			ioutil.WriteFile(path.Join(userDir, ordToFilename[f.Name]), mediaFileContent, 666)
			usn++

			media := db.Media{
				Username: sesh.Username,
			}
			media.Fname = ordToFilename[f.Name]
			media.Usn = usn
			media.Csum = checksum(mediaFileContent)

			filesToAdd = append(filesToAdd, media)

			// write media to media db
			db.DB.Create(&media)
		}

		// TODO: only for debugging with ngrok
		//time.Sleep(5 * time.Second)

		processedFiles := len(filesToRemove) + len(filesToAdd)

		if len(metaFiles) != processedFiles {
			c.String(400, "Sanity check failed")
			return
		}

		// delete removed files
		for _, f := range filesToRemove {
			os.Remove(path.Join(auth.GetUserDir(sesh.Username), f))
		}

		if getLastMediaUsn() != oldUsn+processedFiles {
			c.String(500, "Sanity check failed")
			return
		}
		c.JSON(200, struct {
			Data  []int  `json:"data"`
			Error string `json:"error"`
		}{
			Data:  []int{processedFiles, usn},
			Error: "",
		})

	})
	r.POST("/msync/mediaSanity", sessionMiddleware, func(c *gin.Context) {
		s, _ := c.Get("session")
		sesh, ok := s.(db.Session)
		if !ok {
			c.String(401, "Malformed Session")
			return
		}

		data := getData(c)
		t := struct {
			Local int64 `json:"local"`
		}{}
		json.Unmarshal(data, &t)
		remoteMediaCount := t.Local

		var ownMediaCount int64 = 0
		db.DB.Find(&db.Media{}, "username = ?", sesh.Username).Count(&ownMediaCount)

		if ownMediaCount == remoteMediaCount {
			c.JSON(200, struct {
				Data string `json:"data"`
				Err  string `json:"error"`
			}{
				"OK",
				"",
			})
		} else {
			c.JSON(200, struct {
				Data string `json:"data"`
				Err  string `json:"error"`
			}{
				"FAILED",
				"",
			})
		}

	})

	r.POST("/sync/download", sessionMiddleware, func(c *gin.Context) {
		s, _ := c.Get("session")
		sesh, ok := s.(db.Session)
		if !ok {
			c.String(401, "Malformed Session")
			return
		}

		// create tmp database
		f, err := ioutil.TempFile("", "anki-sync-go")
		if err != nil {
			// TODO: return http error code
			return
		}
		defer os.Remove(f.Name())

		sqlite, err := gorm.Open(sqlite2.Open(f.Name()))
		if err != nil {
			// TODO: actual error handling???
			log.Println(err)
		}

		sqlite.AutoMigrate(&db.SQLiteCol{})
		sqlite.AutoMigrate(&db.SQLiteCard{})
		sqlite.AutoMigrate(&db.SQLiteNote{})
		sqlite.AutoMigrate(&db.SQLiteRevlog{})
		sqlite.AutoMigrate(&db.SQLiteMedia{})
		sqlite.AutoMigrate(&db.SQLiteGraves{})

		var cards []db.SQLiteCard
		var col db.SQLiteCol

		db.DB.Find(&cards, "username = ?", sesh.Username)
		sqlite.Create(&cards)

		db.DB.Find(&col, "username = ?", sesh.Username)
		sqlite.Create(&col)

		var notes []db.SQLiteNote

		db.DB.Find(&notes, "username = ?", sesh.Username)
		sqlite.Create(&notes)

		var graves []db.SQLiteGraves

		db.DB.Find(&graves, "username = ?", sesh.Username)
		sqlite.Create(&graves)

		var revlogs []db.SQLiteRevlog
		db.DB.Find(&revlogs, "username = ?", sesh.Username)

		sqlite.CreateInBatches(&revlogs, 1000)

		data, err := ioutil.ReadAll(f)
		if err != nil {
			c.String(500, "Error when syncing")
			return
		}
		c.String(200, string(data))
	})

	r.POST("/msync/downloadFiles", sessionMiddleware, func(c *gin.Context) {
		s, _ := c.Get("session")
		sesh, ok := s.(db.Session)
		if !ok {
			c.String(401, "Malformed Session")
			return
		}

		data := getData(c)

		requestedFiles := struct {
			Files []string `json:"files"`
		}{}

		// used to write _meta file
		//
		// format:
		// i: filename
		fileList := map[string]string{}

		if err := json.Unmarshal(data, &requestedFiles); err != nil {
			c.String(500, "Could not parse given file list")
			return
		}

		tmpZip, err := ioutil.TempFile("", "anki-sync-go-media")
		if err != nil {
			c.String(500, "Could not create media zip file")
			return
		}
		defer os.Remove(tmpZip.Name())
		zipWriter := zip.NewWriter(tmpZip)
		defer zipWriter.Close()

		userDir := auth.GetUserDir(sesh.Username)
		if userDir == "" {
			c.String(400, "No media data found for this user")
			return
		}

		for i, requestedFileName := range requestedFiles.Files {
			mediaFile, err := os.Open(path.Join(userDir, requestedFileName))
			if err != nil {
				c.String(500, "Could not find requested media file: ", requestedFileName)
				return
			}
			defer mediaFile.Close()

			zipMediaFile, err := zipWriter.Create(strconv.FormatInt(int64(i), 10))
			if err != nil {
				c.String(500, "Could not create requested mediaFile in zip: ", requestedFileName)
				return
			}
			if _, err := io.Copy(zipMediaFile, mediaFile); err != nil {
				c.String(500, "Could not copy requested mediaFile to zip: ", requestedFileName)
				return
			}

			fileList[strconv.FormatInt(int64(i), 10)] = requestedFileName
		}

		// write meta file:
		zf, err := zipWriter.Create("_meta")
		if err != nil {
			c.String(500, "Error when preparing zip file")
			return
		}
		bytes, err := json.Marshal(&fileList)
		if err != nil {
			c.String(400, "Could not parse file list")
			return
		}

		if _, err := zf.Write(bytes); err != nil {
			c.String(500, "An error occurred while writing the _meta file")
			return
		}
		zipWriter.Close()

		c.File(tmpZip.Name())
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

func sessionMiddleware(c *gin.Context) {

	providedKey := c.Request.FormValue("k")

	if providedKey == "" {
		providedKey = c.Request.FormValue("sk")
	}

	var data db.Session

	db.DB.First(&data, "skey = ?", providedKey)

	if data == (db.Session{}) {
		c.String(401, "Could not find session")
		return
	}
	c.Set("session", data)
	c.Next()
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
