package gofyr

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// LogAudit persists an audit log record into gofyr_audit_logs.
func (a *Admin) LogAudit(userID uint, email, action, datasource, model, recordID, ipAddress, details string) {
	if !a.cfg.AuditLogging {
		return
	}

	primaryDB := a.PrimaryDB()
	if primaryDB == nil {
		return
	}

	entry := GofyrAuditLog{
		UserID:     userID,
		UserEmail:  email,
		Action:     action,
		Datasource: datasource,
		Model:      model,
		RecordID:   recordID,
		IPAddress:  ipAddress,
		Details:    details,
		CreatedAt:  time.Now(),
	}

	// Non-blocking write
	go func(e GofyrAuditLog) {
		_ = primaryDB.Create(&e).Error
	}(entry)
}

// handleListAuditLogs returns paginated audit records for the admin UI.
func (a *Admin) handleListAuditLogs(c *gin.Context) {
	primaryDB := a.PrimaryDB()
	if primaryDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Primary datasource unavailable"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))
	if limit < 1 || limit > 100 {
		limit = 25
	}
	offset := (page - 1) * limit

	query := primaryDB.Model(&GofyrAuditLog{})

	if search := c.Query("search"); search != "" {
		s := "%" + search + "%"
		query = query.Where("user_email LIKE ? OR action LIKE ? OR model LIKE ? OR details LIKE ?", s, s, s, s)
	}

	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}

	var total int64
	query.Count(&total)

	var logs []GofyrAuditLog
	if err := query.Order("id desc").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch audit logs: %v", err)})
		return
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, gin.H{
		"data":        logs,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}
