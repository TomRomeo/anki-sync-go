package main

import (
	"ankiSyncGo/internal/auth"
	"ankiSyncGo/internal/db"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// parse dotEnv if present
	_ = godotenv.Load()

	// handle db
	db.Initialize()

	r := gin.Default()
	r.Use(sessions.Sessions("sessions", db.Store))
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

		file, _, _ := c.Request.FormFile("data")
		var b []byte

		// decompress if compressed
		gr, _ := gzip.NewReader(file)
		b, _ = ioutil.ReadAll(gr)

		var data struct {
			U string `json:"u"` // Username
			P string `json:"p"` // Password
		}

		if err := json.Unmarshal(b, &data); err != nil {
			_ = c.AbortWithError(400, errors.New("Did not find username and password"))
			return
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

		session := sessions.Default(c)

		// TODO: generate actual key
		session.Set("skey", "12345")
		session.Set("name", data.U)

		if err := session.Save(); err != nil {
			log.Fatal(err)
		}

		c.JSON(200, struct {
			Key string `json:"key"`
		}{"12345"})
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
