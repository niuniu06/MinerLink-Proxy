package ui

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var distFS embed.FS

// RegisterUI adds the embedded Vue 3 frontend to the gin router
func RegisterUI(r *gin.Engine) {
	// Extract the "dist" directory from the embed.FS
	subFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}

	// Serve static files but disable caching for index.html
	r.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/ui/" || c.Request.URL.Path == "/ui/index.html" {
			c.Writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Writer.Header().Set("Pragma", "no-cache")
			c.Writer.Header().Set("Expires", "0")
		}
		c.Next()
	})

	r.StaticFS("/ui", http.FS(subFS))
	
	// Redirect root to /ui/
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui/")
	})
}
