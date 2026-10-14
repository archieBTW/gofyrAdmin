package gofyr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RouteInfo represents an introspected Gin route.
type RouteInfo struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Handler     string `json:"handler"`
	Group       string `json:"group"`
	IsAdmin     bool   `json:"is_admin"`
	Description string `json:"description,omitempty"`
}

// WorkbenchExecuteRequest represents the payload to execute an API call.
type WorkbenchExecuteRequest struct {
	Method      string            `json:"method" binding:"required"`
	Path        string            `json:"path" binding:"required"`
	QueryParams map[string]string `json:"query_params"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
}

// WorkbenchExecuteResponse contains execution results and metrics.
type WorkbenchExecuteResponse struct {
	StatusCode int               `json:"status_code"`
	StatusText string            `json:"status_text"`
	DurationMS float64           `json:"duration_ms"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	IsJSON     bool              `json:"is_json"`
}

// IntrospectRoutes retrieves all routes from the host Gin engine (or target engine).
func (a *Admin) IntrospectRoutes() []RouteInfo {
	engine := a.hostEngine
	if a.targetEngine != nil {
		engine = a.targetEngine
	}
	if engine == nil {
		return nil
	}

	routes := engine.Routes()
	var result []RouteInfo

	for _, r := range routes {
		isAdmin := strings.HasPrefix(r.Path, a.cfg.BasePath)

		// Determine a clean group name
		parts := strings.Split(strings.Trim(r.Path, "/"), "/")
		group := "Root"
		if len(parts) > 0 && parts[0] != "" {
			group = "/" + parts[0]
			if len(parts) > 1 && parts[0] == "api" {
				group = "/api/" + parts[1]
			}
		}

		cleanHandler := r.Handler
		if idx := strings.LastIndex(cleanHandler, "/"); idx != -1 {
			cleanHandler = cleanHandler[idx+1:]
		}

		result = append(result, RouteInfo{
			Method:  r.Method,
			Path:    r.Path,
			Handler: cleanHandler,
			Group:   group,
			IsAdmin: isAdmin,
		})
	}

	// Also introspect custom dynamic routes created via UI
	primaryDB := a.PrimaryDB()
	if primaryDB != nil {
		var customRoutes []GofyrCustomRoute
		_ = primaryDB.Find(&customRoutes).Error
		for _, cr := range customRoutes {
			// Avoid duplicates if already registered in host routes
			found := false
			for _, r := range result {
				if r.Method == cr.Method && r.Path == cr.Path {
					found = true
					break
				}
			}
			if !found {
				result = append(result, RouteInfo{
					Method:      cr.Method,
					Path:        cr.Path,
					Handler:     "custom.mock.handler",
					Group:       "Custom Mock",
					IsAdmin:     false,
					Description: cr.Description,
				})
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].IsAdmin != result[j].IsAdmin {
			return !result[i].IsAdmin // Host routes first
		}
		if result[i].Group != result[j].Group {
			return result[i].Group < result[j].Group
		}
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		return result[i].Method < result[j].Method
	})

	return result
}

// handleListWorkbenchRoutes handles GET <BasePath>/api/workbench/routes
func (a *Admin) handleListWorkbenchRoutes(c *gin.Context) {
	routes := a.IntrospectRoutes()
	c.JSON(http.StatusOK, gin.H{
		"routes": routes,
		"count":  len(routes),
	})
}

// CreateCustomRouteRequest defines payload to create a custom dynamic mock route from the UI.
type CreateCustomRouteRequest struct {
	Method       string `json:"method" binding:"required"`
	Path         string `json:"path" binding:"required"`
	StatusCode   int    `json:"status_code"`
	HeadersJSON  string `json:"headers_json"`
	ResponseBody string `json:"response_body"`
	Description  string `json:"description"`
}

// handleCreateCustomRoute creates and registers a dynamic route directly from the Workbench UI.
func (a *Admin) handleCreateCustomRoute(c *gin.Context) {
	var req CreateCustomRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	req.Method = strings.ToUpper(strings.TrimSpace(req.Method))
	req.Path = strings.TrimSpace(req.Path)
	if !strings.HasPrefix(req.Path, "/") {
		req.Path = "/" + req.Path
	}
	if req.StatusCode == 0 {
		req.StatusCode = http.StatusOK
	}

	primaryDB := a.PrimaryDB()
	if primaryDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Primary datasource unavailable"})
		return
	}

	rec := GofyrCustomRoute{
		Method:       req.Method,
		Path:         req.Path,
		StatusCode:   req.StatusCode,
		HeadersJSON:  req.HeadersJSON,
		ResponseBody: req.ResponseBody,
		Description:  req.Description,
		CreatedAt:    time.Now(),
	}

	if err := primaryDB.Create(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save custom route: " + err.Error()})
		return
	}

	// Register route on host Gin engine if not already registered
	if a.hostEngine != nil {
		a.hostEngine.Handle(rec.Method, rec.Path, func(ctx *gin.Context) {
			if rec.HeadersJSON != "" {
				var hdrs map[string]string
				if err := json.Unmarshal([]byte(rec.HeadersJSON), &hdrs); err == nil {
					for k, v := range hdrs {
						ctx.Header(k, v)
					}
				}
			}
			contentType := "application/json; charset=utf-8"
			if !strings.HasPrefix(strings.TrimSpace(rec.ResponseBody), "{") && !strings.HasPrefix(strings.TrimSpace(rec.ResponseBody), "[") {
				contentType = "text/plain; charset=utf-8"
			}
			ctx.Data(rec.StatusCode, contentType, []byte(rec.ResponseBody))
		})
	}

	user := a.CurrentUser(c)
	userEmail := "system"
	var userID uint
	if user != nil {
		userID = user.UserID
		userEmail = user.Email
	}
	a.LogAudit(userID, userEmail, "route_create", "", "CustomRoute", rec.Path, c.ClientIP(), fmt.Sprintf("Created custom route %s %s", rec.Method, rec.Path))

	c.JSON(http.StatusCreated, gin.H{
		"message": "Custom route created and mounted successfully",
		"route":   rec,
	})
}

// handleDeleteCustomRoute removes a custom route.
func (a *Admin) handleDeleteCustomRoute(c *gin.Context) {
	id := c.Param("id")
	primaryDB := a.PrimaryDB()
	if primaryDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Primary datasource unavailable"})
		return
	}

	if err := primaryDB.Delete(&GofyrCustomRoute{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete route: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Custom route deleted successfully"})
}

// LoadCustomRoutes mounts any persistent custom routes from the database.
func (a *Admin) LoadCustomRoutes() error {
	primaryDB := a.PrimaryDB()
	if primaryDB == nil || a.hostEngine == nil {
		return nil
	}

	var customRoutes []GofyrCustomRoute
	if err := primaryDB.Find(&customRoutes).Error; err != nil {
		return err
	}

	for _, cr := range customRoutes {
		rec := cr
		a.hostEngine.Handle(rec.Method, rec.Path, func(ctx *gin.Context) {
			if rec.HeadersJSON != "" {
				var hdrs map[string]string
				if err := json.Unmarshal([]byte(rec.HeadersJSON), &hdrs); err == nil {
					for k, v := range hdrs {
						ctx.Header(k, v)
					}
				}
			}
			contentType := "application/json; charset=utf-8"
			if !strings.HasPrefix(strings.TrimSpace(rec.ResponseBody), "{") && !strings.HasPrefix(strings.TrimSpace(rec.ResponseBody), "[") {
				contentType = "text/plain; charset=utf-8"
			}
			ctx.Data(rec.StatusCode, contentType, []byte(rec.ResponseBody))
		})
	}

	return nil
}

// handleExecuteWorkbenchRoute handles POST <BasePath>/api/workbench/execute
func (a *Admin) handleExecuteWorkbenchRoute(c *gin.Context) {
	engine := a.hostEngine
	if a.targetEngine != nil {
		engine = a.targetEngine
	}
	if engine == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Target Gin engine is not available"})
		return
	}

	var req WorkbenchExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Prepare URL with query parameters
	parsedURL, err := url.Parse(req.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path: " + err.Error()})
		return
	}

	q := parsedURL.Query()
	for k, v := range req.QueryParams {
		q.Set(k, v)
	}
	parsedURL.RawQuery = q.Encode()

	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = bytes.NewBufferString(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), strings.ToUpper(req.Method), parsedURL.String(), bodyReader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request: " + err.Error()})
		return
	}

	// Set headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	if req.Body != "" && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Execute through host Gin router using httptest.ResponseRecorder (in-process, zero network lag)
	recorder := httptest.NewRecorder()
	start := time.Now()
	a.hostEngine.ServeHTTP(recorder, httpReq)
	duration := time.Since(start)

	respHeaders := make(map[string]string)
	for k, vals := range recorder.Header() {
		if len(vals) > 0 {
			respHeaders[k] = vals[0]
		}
	}

	respBody := recorder.Body.String()
	contentType := recorder.Header().Get("Content-Type")
	isJSON := strings.Contains(contentType, "application/json")

	c.JSON(http.StatusOK, WorkbenchExecuteResponse{
		StatusCode: recorder.Code,
		StatusText: http.StatusText(recorder.Code),
		DurationMS: float64(duration.Microseconds()) / 1000.0,
		Headers:    respHeaders,
		Body:       respBody,
		IsJSON:     isJSON,
	})
}
