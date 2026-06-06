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

	r.StaticFS("/ui", http.FS(subFS))
	
	// Redirect root to /ui/
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui/")
	})
}
