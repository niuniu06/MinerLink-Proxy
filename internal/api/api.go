package api

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	
	"proxy-core/internal/db"
	"proxy-core/internal/models"
	"proxy-core/internal/proxy"
	"proxy-core/internal/ui"
)

type APIServer struct {
	ProxyManager *proxy.Manager
}

func NewAPIServer(pm *proxy.Manager) *APIServer {
	return &APIServer{
		ProxyManager: pm,
	}
}

func (s *APIServer) Start(port int) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.GET("/stats", s.getStats)
		api.GET("/config", s.getConfig)
		api.POST("/config/add", s.addConfig)
		api.POST("/config/delete", s.deleteConfig)
		api.POST("/system/restart", s.restartSystem)

		api.GET("/global", s.getGlobalConfig)
		api.POST("/global", s.saveGlobalConfig)
	}

	ui.RegisterUI(r)

	return r.Run(":" + strconv.Itoa(port))
}

func (s *APIServer) getStats(c *gin.Context) {
	stats := make([]map[string]interface{}, 0)
	s.ProxyManager.Servers.Range(func(key, value interface{}) bool {
		server := value.(*proxy.Server)
		stats = append(stats, server.GetStats())
		return true
	})
	c.JSON(http.StatusOK, stats)
}

func (s *APIServer) getConfig(c *gin.Context) {
	configs, err := db.GetAllConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}

func (s *APIServer) addConfig(c *gin.Context) {
	var cfg models.ProxyConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.SaveConfig(&cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Hot reload
	s.ProxyManager.RestartProxy(cfg.ListenPort)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Config saved and proxy restarted"})
}

func (s *APIServer) deleteConfig(c *gin.Context) {
	var req struct {
		ListenPort int `json:"listenPort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.DeleteConfig(req.ListenPort); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Stop proxy
	s.ProxyManager.StopProxy(req.ListenPort)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *APIServer) restartSystem(c *gin.Context) {
	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *APIServer) getGlobalConfig(c *gin.Context) {
	cfg, err := db.GetGlobalConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (s *APIServer) saveGlobalConfig(c *gin.Context) {
	var cfg models.GlobalConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.SaveGlobalConfig(&cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
	
	// Exit and let systemd automatically restart to apply new port
	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
}
