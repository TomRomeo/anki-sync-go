package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.String(200, "Anki Sync server written in Go 🚀")
	})
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.File("favicon.svg")
	})
	r.Run("0.0.0.0:27701")
}
