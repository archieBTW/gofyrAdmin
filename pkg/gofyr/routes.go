package gofyr

import (
	"bytes"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/archieBTW/gofyr/pkg/gofyr/storage"
	"github.com/archieBTW/gofyr/pkg/gofyr/ui"
)

// Mount mounts gofyrAdmin routes and embedded UI onto the target Gin engine.
func (a *Admin) Mount(r *gin.Engine) error {
	a.hostEngine = r

	base := a.cfg.BasePath

	// Mount local storage static files if local driver is active
	if localDriver, ok := a.storageDrivers["local"]; ok {
		if ld, ok := localDriver.(*storage.Local); ok {
			prefix := ld.PublicURLPrefix()
			r.StaticFS(prefix, http.Dir(ld.BaseDir()))
		}
	}

	// API Group
	api := r.Group(base + "/api")
	{
		// Public Auth routes
		api.POST("/auth/login", a.handleLogin)
		api.POST("/auth/mfa/setup", a.handleMFASetup)
		api.POST("/auth/mfa/verify", a.handleMFAVerify)

		// Protected routes
		protected := api.Group("")
		protected.Use(a.AuthMiddleware())
		{
			// Current user & session
			protected.GET("/auth/me", a.handleMe)
			protected.POST("/auth/logout", a.handleLogout)

			// Schema
			protected.GET("/schema", a.handleGetSchema)
			protected.POST("/schema/models", a.handleCreateCustomModel)

			// Audit logs
			protected.GET("/audit-logs", a.handleListAuditLogs)

			// Dynamic CRUD
			records := protected.Group("/ds/:datasource/records/:model")
			{
				records.GET("", a.handleListRecords)
				records.GET("/:id", a.handleGetRecord)
				records.POST("", a.handleCreateRecord)
				records.PUT("/:id", a.handleUpdateRecord)
				records.DELETE("/:id", a.handleDeleteRecord)
				records.POST("/bulk-delete", a.handleBulkDeleteRecords)
			}

			// Media Library
			media := protected.Group("/media")
			{
				media.POST("/upload", a.handleMediaUpload)
				media.GET("/files", a.handleListMediaFiles)
				media.DELETE("/files/:id", a.handleDeleteMediaFile)
			}

			// API Workbench
			workbench := protected.Group("/workbench")
			{
				workbench.GET("/routes", a.handleListWorkbenchRoutes)
				workbench.POST("/execute", a.handleExecuteWorkbenchRoute)
				workbench.POST("/custom-routes", a.handleCreateCustomRoute)
				workbench.DELETE("/custom-routes/:id", a.handleDeleteCustomRoute)
			}
		}
	}

	// Load dynamic custom models and custom routes from primary DB
	_ = a.LoadCustomModels()
	_ = a.LoadCustomRoutes()

	// Mount Embedded Svelte UI
	subFS, err := fs.Sub(ui.DistFS, "dist")
	if err != nil {
		return fmt.Errorf("gofyr: failed to load embedded dist fs: %w", err)
	}

	indexHTML, err := fs.ReadFile(subFS, "index.html")
	if err != nil {
		indexHTML = []byte(`<!DOCTYPE html><html><head><title>gofyrAdmin</title></head><body><div id="app">Loading gofyrAdmin...</div></body></html>`)
	}

	// Inject runtime config into index.html
	injectedHTML := injectConfigHTML(indexHTML, a.cfg.BasePath, a.cfg.Title)

	// UI handler function
	uiHandler := func(c *gin.Context) {
		reqPath := strings.TrimPrefix(c.Request.URL.Path, base)
		reqPath = strings.TrimPrefix(reqPath, "/")

		// Don't intercept API calls
		if strings.HasPrefix(reqPath, "api") {
			c.Next()
			return
		}

		// Try serving static file if path is not empty and not an SPA route
		if reqPath != "" && reqPath != "index.html" {
			file, err := subFS.Open(reqPath)
			if err == nil {
				defer file.Close()
				stat, err := file.Stat()
				if err == nil && !stat.IsDir() {
					contentType := mime.TypeByExtension(filepath.Ext(reqPath))
					if contentType == "" {
						contentType = "application/octet-stream"
					}
					c.DataFromReader(http.StatusOK, stat.Size(), contentType, file, nil)
					return
				}
			}
		}

		// Fallback to index.html for SPA client-side routing
		c.Data(http.StatusOK, "text/html; charset=utf-8", injectedHTML)
	}

	r.GET(base, uiHandler)
	r.GET(base+"/", uiHandler)

	// Mount assets subfolder directly
	if assetsSubFS, err := fs.Sub(subFS, "assets"); err == nil {
		r.StaticFS(base+"/assets", http.FS(assetsSubFS))
	}

	// Use NoRoute for SPA route fallback
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, base) && !strings.HasPrefix(c.Request.URL.Path, base+"/api") {
			uiHandler(c)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Page not found"})
	})

	return nil
}

// handleGetSchema handles GET <BasePath>/api/schema
func (a *Admin) handleGetSchema(c *gin.Context) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var dsSchemas []DatasourceSchema
	for _, ds := range a.datasources {
		ds.mu.RLock()
		var modelsCopy []ModelInfo
		for _, m := range ds.Models {
			modelsCopy = append(modelsCopy, *m)
		}
		ds.mu.RUnlock()

		dsSchemas = append(dsSchemas, DatasourceSchema{
			Name:      ds.Name,
			Dialect:   ds.Dialect,
			IsPrimary: ds.IsPrimary,
			Models:    modelsCopy,
		})
	}

	var diskNames []string
	for name := range a.storageDrivers {
		diskNames = append(diskNames, name)
	}

	resp := SchemaResponse{
		Title:       a.cfg.Title,
		BasePath:    a.cfg.BasePath,
		DefaultDisk: a.cfg.DefaultDisk,
		Disks:       diskNames,
		Datasources: dsSchemas,
	}

	c.JSON(http.StatusOK, resp)
}

func injectConfigHTML(orig []byte, basePath, title string) []byte {
	configScript := fmt.Sprintf(`<script>window.__GOFYR_CONFIG__ = { basePath: "%s", title: "%s" };</script>`, basePath, title)
	target := []byte("<head>")
	if bytes.Contains(orig, target) {
		return bytes.Replace(orig, target, append(target, []byte("\n"+configScript)...), 1)
	}
	return append([]byte(configScript), orig...)
}
