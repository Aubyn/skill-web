package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"skill-web/db"
	"skill-web/handler"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed frontend/dist/*
var frontendFS embed.FS

func main() {
	// Initialize database
	database, err := db.Open()
	if err != nil {
		panic("db: " + err.Error())
	}
	defer database.Close()

	// Store directory
	home, _ := os.UserHomeDir()
	storeDir := filepath.Join(home, ".skill-web", "skills")
	os.MkdirAll(storeDir, 0755)

	// Echo server
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}))

	// API routes
	sh := handler.NewSkillHandler(database, storeDir)
	gh := handler.NewGroupHandler(database)
	th := handler.NewTargetHandler(database)
	syh := handler.NewSyncHandler(database, storeDir)

	api := e.Group("/api")
	{
		// Skills
		api.GET("/skills", sh.List)
		api.GET("/skills/:id", sh.Get)
		api.DELETE("/skills/:id", sh.Delete)
		api.GET("/scan-preview", sh.ScanPreview)
		api.POST("/import", sh.Import)

		// Groups
		api.GET("/groups", gh.List)
		api.POST("/groups", gh.Create)
		api.GET("/groups/:id", gh.Get)
		api.DELETE("/groups/:id", gh.Delete)
		api.POST("/groups/:id/skills", gh.AddSkills)
		api.DELETE("/groups/:id/skills", gh.RemoveSkills)

		// Targets
		api.GET("/targets", th.List)
		api.POST("/targets", th.Create)
		api.DELETE("/targets/:id", th.Delete)
		api.GET("/targets/:id/skills", th.ListSkills)
		api.POST("/targets/:id/clear", th.Clear)

		// Sync
		api.POST("/sync", syh.Sync)
	}

	// Static files from embedded frontend
	staticFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		panic("embed fs: " + err.Error())
	}

	// SPA handler: serve index.html for all non-API routes
	staticHandler := http.FileServer(http.FS(staticFS))
	e.GET("/*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the exact file
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists in embedded FS
		f, err := staticFS.Open(path[1:]) // strip leading /
		if err == nil {
			f.Close()
			staticHandler.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html
		r.URL.Path = "/"
		staticHandler.ServeHTTP(w, r)
	})))

	// Try ports 7931-7940 until one is free
	var listener net.Listener
	for port := 7931; port <= 7940; port++ {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		var err error
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			e.Logger.Warnf("port %d unavailable, trying next", port)
			continue
		}
		fmt.Printf("skill-web running on http://%s\n", addr)
		break
	}
	if listener == nil {
		e.Logger.Fatal("no available port found in range 7931-7940")
	}
	e.Listener = listener
	e.Server.Handler = e
	if err := e.Server.Serve(listener); err != nil {
		e.Logger.Fatal(err)
	}
}
