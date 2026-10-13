package gofyr

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// handleMediaUpload processes multipart file uploads to the designated storage driver.
func (a *Admin) handleMediaUpload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded: " + err.Error()})
		return
	}

	diskName := c.DefaultPostForm("disk", a.cfg.DefaultDisk)
	if diskName == "" {
		diskName = "local"
	}

	folder := c.DefaultPostForm("folder", "general")
	folder = strings.Trim(folder, "/")

	driver, ok := a.storageDrivers[diskName]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Storage driver '%s' is not registered", diskName)})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read uploaded file: " + err.Error()})
		return
	}
	defer file.Close()

	// Detect MIME type
	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	mimeType := http.DetectContentType(buffer[:n])
	// Reset file reader to beginning
	if _, err := file.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset file stream: " + err.Error()})
		return
	}

	// Generate clean filename
	origName := filepath.Base(fileHeader.Filename)
	ext := filepath.Ext(origName)
	base := strings.TrimSuffix(origName, ext)
	// Sanitize base
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, base)
	cleanFilename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano()/1e6, base, ext)
	key := filepath.Join(folder, cleanFilename)
	key = strings.ReplaceAll(key, "\\", "/")

	publicURL, err := driver.Upload(c.Request.Context(), key, file, mimeType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Storage driver upload failed: " + err.Error()})
		return
	}

	mediaFile := GofyrMediaFile{
		Disk:         diskName,
		Folder:       folder,
		Filename:     cleanFilename,
		OriginalName: origName,
		MimeType:     mimeType,
		Size:         fileHeader.Size,
		URL:          publicURL,
		CreatedAt:    time.Now(),
	}

	primaryDB := a.PrimaryDB()
	if primaryDB != nil {
		primaryDB.Create(&mediaFile)
	}

	user := a.CurrentUser(c)
	userEmail := "system"
	var userID uint
	if user != nil {
		userID = user.UserID
		userEmail = user.Email
	}
	a.LogAudit(userID, userEmail, "media_upload", "", "GofyrMediaFile", fmt.Sprintf("%d", mediaFile.ID), c.ClientIP(), fmt.Sprintf("Uploaded %s to %s (%s)", origName, diskName, publicURL))

	c.JSON(http.StatusOK, gin.H{
		"data": mediaFile,
	})
}

// handleListMediaFiles returns paginated media library records.
func (a *Admin) handleListMediaFiles(c *gin.Context) {
	primaryDB := a.PrimaryDB()
	if primaryDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Primary datasource unavailable"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))
	if limit < 1 || limit > 100 {
		limit = 24
	}
	offset := (page - 1) * limit

	query := primaryDB.Model(&GofyrMediaFile{})

	if disk := c.Query("disk"); disk != "" {
		query = query.Where("disk = ?", disk)
	}

	if search := c.Query("search"); search != "" {
		s := "%" + search + "%"
		query = query.Where("filename LIKE ? OR original_name LIKE ? OR folder LIKE ?", s, s, s)
	}

	if mimeFilter := c.Query("type"); mimeFilter != "" {
		if mimeFilter == "image" {
			query = query.Where("mime_type LIKE ?", "image/%")
		} else if mimeFilter == "doc" {
			query = query.Where("mime_type NOT LIKE ?", "image/%")
		}
	}

	var total int64
	query.Count(&total)

	var files []GofyrMediaFile
	if err := query.Order("id desc").Offset(offset).Limit(limit).Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to list media files: %v", err)})
		return
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, gin.H{
		"data":        files,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

// handleDeleteMediaFile removes a file from the storage driver and database.
func (a *Admin) handleDeleteMediaFile(c *gin.Context) {
	id := c.Param("id")
	primaryDB := a.PrimaryDB()
	if primaryDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Primary datasource unavailable"})
		return
	}

	var mediaFile GofyrMediaFile
	if err := primaryDB.First(&mediaFile, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media file not found"})
		return
	}

	// Delete from storage driver if driver is available
	if driver, ok := a.storageDrivers[mediaFile.Disk]; ok {
		key := filepath.Join(mediaFile.Folder, mediaFile.Filename)
		key = strings.ReplaceAll(key, "\\", "/")
		_ = driver.Delete(c.Request.Context(), key)
	}

	if err := primaryDB.Delete(&mediaFile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file record: " + err.Error()})
		return
	}

	user := a.CurrentUser(c)
	userEmail := "system"
	var userID uint
	if user != nil {
		userID = user.UserID
		userEmail = user.Email
	}
	a.LogAudit(userID, userEmail, "media_delete", "", "GofyrMediaFile", id, c.ClientIP(), fmt.Sprintf("Deleted file %s from %s", mediaFile.Filename, mediaFile.Disk))

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}
