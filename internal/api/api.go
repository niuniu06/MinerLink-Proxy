package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"proxy-core/internal/db"
	"proxy-core/internal/logger"
	"proxy-core/internal/models"
	"proxy-core/internal/proxy"
	"proxy-core/internal/ui"
	"proxy-core/internal/sysinfo"
	"proxy-core/internal/updater"
	"strings"
	"crypto/md5"
)

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")
		
		globalCfg, err := db.GetGlobalConfig()
		if err != nil || globalCfg == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		
		expectedToken := fmt.Sprintf("%x", md5.Sum([]byte("MLP:"+globalCfg.AdminAccount+":"+globalCfg.AdminPassword)))
		
		if token != expectedToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		c.Next()
	}
}

//go:embed downloads/*
var embedDownloadsFS embed.FS

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

	r.POST("/api/login", s.login)

	api := r.Group("/api")
	api.Use(authMiddleware())
	{
		api.GET("/stats", s.getStats)
		api.GET("/config", s.getConfig)
		api.POST("/config/add", s.addConfig)
		api.POST("/config/delete", s.deleteConfig)

		api.POST("/system/restart", s.restartSystem)
		api.POST("/system/ping", s.pingPool)
		api.GET("/system/status", s.getSystemStatus)
		api.GET("/system/check_update", s.checkUpdate)
		api.POST("/system/upgrade", s.doUpgrade)

		api.GET("/global", s.getGlobalConfig)
		api.POST("/config/save", s.saveGlobalConfig)
		api.GET("/logs/tail", s.getLogs)
		api.DELETE("/logs/clear", s.clearLogs)
		
		api.GET("/miners", s.getMiners)
		api.GET("/minerlogs", s.getMinerLogs)
		api.POST("/download/custom", s.downloadCustomClient)
		api.GET("/download/custom", s.downloadCustomClient)
	}

	// Serve static files for downloads (e.g. tunnel clients)
	if _, err := os.Stat("./downloads"); os.IsNotExist(err) {
		_ = os.Mkdir("./downloads", 0755)
	}
	r.StaticFS("/downloads", http.Dir("./downloads"))

	ui.RegisterUI(r)

	return r.Run(":" + strconv.Itoa(port))
}

func (s *APIServer) login(c *gin.Context) {
	var req struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	globalCfg, _ := db.GetGlobalConfig()
	if globalCfg == nil || req.Account != globalCfg.AdminAccount || req.Password != globalCfg.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	expectedToken := fmt.Sprintf("%x", md5.Sum([]byte("MLP:"+globalCfg.AdminAccount+":"+globalCfg.AdminPassword)))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"token":   expectedToken,
	})
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

func (s *APIServer) getMiners(c *gin.Context) {
	portStr := c.Query("port")
	pageStr := c.Query("page")
	limitStr := c.Query("limit")
	searchStr := strings.ToLower(c.Query("search"))
	
	port, _ := strconv.Atoi(portStr)
	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 12
	}

	if port == 0 {
		// Global miners list
		allMiners := s.ProxyManager.GetAllMiners()
		
		// Apply search filter if present
		if searchStr != "" {
			filtered := make([]map[string]interface{}, 0)
			for _, m := range allMiners {
				worker := ""
				if w, ok := m["worker"].(string); ok {
					worker = strings.ToLower(w)
				}
				if strings.Contains(worker, searchStr) {
					filtered = append(filtered, m)
				}
			}
			allMiners = filtered
		}
		
		total := len(allMiners)
		start := (page - 1) * limit
		end := start + limit
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}
		
		c.JSON(http.StatusOK, gin.H{
			"total": total,
			"page": page,
			"limit": limit,
			"miners": allMiners[start:end],
		})
		return
	}
	
	val, ok := s.ProxyManager.Servers.Load(port)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "proxy not found on this port"})
		return
	}
	server := val.(*proxy.Server)
	
	total, paginatedMiners := server.GetPaginatedMiners(page, limit)
	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page": page,
		"limit": limit,
		"miners": paginatedMiners,
	})
}

func (s *APIServer) getMinerLogs(c *gin.Context) {
	portStr := c.Query("port")
	worker := c.Query("worker")
	
	port, _ := strconv.Atoi(portStr)
	val, ok := s.ProxyManager.Servers.Load(port)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "proxy not found on this port"})
		return
	}
	server := val.(*proxy.Server)
	
	genLogs, errLogs := server.GetMinerLogs(worker)
	c.JSON(http.StatusOK, gin.H{
		"general": genLogs,
		"error": errLogs,
	})
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
	var req struct {
		models.ProxyConfig
		IsEdit        bool `json:"isEdit"`
		OldListenPort int  `json:"oldListenPort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg := req.ProxyConfig

	cfg.PoolAddress = strings.TrimSpace(cfg.PoolAddress)
	cfg.FeePoolAddress = strings.TrimSpace(cfg.FeePoolAddress)

	// Strip URL schemes from PoolAddress and FeePoolAddress
	prefixes := []string{"stratum+tcp://", "stratum+ssl://", "tcp://", "ssl://", "http://", "https://"}
	for _, p := range prefixes {
		cfg.PoolAddress = strings.TrimPrefix(cfg.PoolAddress, p)
		cfg.FeePoolAddress = strings.TrimPrefix(cfg.FeePoolAddress, p)
	}



	// Check if this is a NEW config or a port modification
	configs, _ := db.GetAllConfigs()
	portExists := false
	for _, exist := range configs {
		if exist.ListenPort == cfg.ListenPort {
			portExists = true
			break
		}
	}

	isPortChanged := req.IsEdit && req.OldListenPort > 0 && req.OldListenPort != cfg.ListenPort

	if !req.IsEdit || isPortChanged {
		if portExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("端口 %d 配置已存在！若要修改请点击编辑按钮，若要新增请更换端口。", cfg.ListenPort)})
			return
		}
		// Verify port is not in use by other software
		if isPortInUse(cfg.ListenPort) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("端口 %d 已被系统其他程序占用，请更换其他端口。", cfg.ListenPort)})
			return
		}
	}

	if isPortChanged {
		// Clean up old port
		_ = db.DeleteConfig(req.OldListenPort)
		s.ProxyManager.StopProxy(req.OldListenPort)
	}

	if err := db.SaveConfig(&cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Hot reload
	if err := s.ProxyManager.RestartProxy(cfg.ListenPort); err != nil {
		// Port bind failed
		errMsg := err.Error()
		if strings.Contains(errMsg, "address already in use") {
			errMsg = "该端口已被其他程序占用 (Address already in use)"
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("配置已保存，但端口 %d 启动失败：%s", cfg.ListenPort, errMsg)})
		return
	}

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


func (s *APIServer) getLogs(c *gin.Context) {
	// Read last 200 lines from proxy.log using os and strings
	logPath := logger.LogFilePath
	content, err := os.ReadFile(logPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Log file not found or unreadable"})
		return
	}
	lines := strings.Split(string(content), "\n")
	tailCount := 300
	if len(lines) < tailCount {
		tailCount = len(lines)
	}
	tailLines := lines[len(lines)-tailCount:]
	c.String(http.StatusOK, strings.Join(tailLines, "\n"))
}

func (s *APIServer) clearLogs(c *gin.Context) {
	logPath := logger.LogFilePath
	err := os.WriteFile(logPath, []byte(""), 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear log file"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *APIServer) restartSystem(c *gin.Context) {
	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *APIServer) pingPool(c *gin.Context) {
	var req struct {
		PoolAddress string `json:"poolAddress"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	addr := req.PoolAddress
	if strings.Contains(addr, "://") {
		parts := strings.SplitN(addr, "://", 2)
		addr = parts[1]
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	latency := time.Since(start)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":   false,
			"latencyMs": 0,
			"error":     err.Error(),
		})
		return
	}
	conn.Close()

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"latencyMs": float64(latency.Microseconds()) / 1000.0,
		"error":     "",
	})
}

func (s *APIServer) downloadCustomClient(c *gin.Context) {
	var osType string
	var mappings []map[string]string

	if c.Request.Method == "POST" {
		var req struct {
			OS       string              `json:"os"`
			Mappings []map[string]string `json:"mappings"`
			Remote   string              `json:"remote"`
			Local    string              `json:"local"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		osType = req.OS
		mappings = req.Mappings
		if len(mappings) == 0 && req.Remote != "" && req.Local != "" {
			mappings = append(mappings, map[string]string{"remote": req.Remote, "local": req.Local})
		}
	} else {
		osType = c.Query("os")
		mappingsStr := c.Query("mappings")
		if mappingsStr != "" {
			_ = json.Unmarshal([]byte(mappingsStr), &mappings)
		} else {
			remote := c.Query("remote")
			local := c.Query("local")
			if remote != "" && local != "" {
				mappings = append(mappings, map[string]string{"remote": remote, "local": local})
			}
		}
	}

	if len(mappings) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing mappings configuration"})
		return
	}

	fileName := "local-tunnel-windows-amd64.exe"
	if osType == "linux" {
		fileName = "local-tunnel-linux-amd64"
	}

	path := "downloads/" + fileName
	baseBytes, err := embedDownloadsFS.ReadFile(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Base client not found"})
		return
	}

	configBytes, _ := json.Marshal(map[string]interface{}{
		"mappings": mappings,
	})

	var outBytes []byte
	outBytes = append(outBytes, baseBytes...)
	outBytes = append(outBytes, configBytes...)

	lengthStr := fmt.Sprintf("%08d", len(configBytes))
	outBytes = append(outBytes, []byte(lengthStr)...)
	outBytes = append(outBytes, []byte("ZSDT_CFG")...)

	downloadName := "go-xy.exe"
	if osType == "linux" {
		downloadName = "go-xy"
	}

	c.Header("Content-Disposition", "attachment; filename="+downloadName)
	c.Data(http.StatusOK, "application/octet-stream", outBytes)
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

	// Get current active web port from the request
	currentActivePort := 80
	if strings.Contains(c.Request.Host, ":") {
		_, portStr, err := net.SplitHostPort(c.Request.Host)
		if err == nil {
			if p, err := strconv.Atoi(portStr); err == nil {
				currentActivePort = p
			}
		}
	}

	// Check if web port is changed and new port is in use (skip check if it's the current active port)
	currentCfg, err := db.GetGlobalConfig()
	if err == nil && currentCfg.WebPort != cfg.WebPort && cfg.WebPort > 0 {
		if cfg.WebPort != currentActivePort && isPortInUse(cfg.WebPort) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("网页端口 %d 已被系统其他程序占用，请更换其他端口！", cfg.WebPort)})
			return
		}
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

// isPortInUse checks if a specific port is already bound on the system
func isPortInUse(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return true // Port is in use or inaccessible
	}
	l.Close()
	return false
}

func (s *APIServer) getSystemStatus(c *gin.Context) {
	status := sysinfo.GetSystemStatus()
	c.JSON(http.StatusOK, status)
}

func (s *APIServer) checkUpdate(c *gin.Context) {
	mock := c.Query("mock")
	if mock == "1" {
		c.JSON(http.StatusOK, gin.H{
			"hasUpdate":      false,
			"currentVersion": sysinfo.ProxyVersion,
			"latestVersion":  sysinfo.ProxyVersion,
			"changelog":      "",
		})
		return
	}

	status, err := updater.CheckForUpdates()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"hasUpdate":      false,
			"currentVersion": sysinfo.ProxyVersion,
			"latestVersion":  "",
			"changelog":      "",
			"error":          err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (s *APIServer) doUpgrade(c *gin.Context) {
	mock := c.Query("mock")
	if mock == "1" {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Mock upgrade started"})
		return
	}

	err := updater.StartUpgrade()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Upgrade started successfully"})
}
