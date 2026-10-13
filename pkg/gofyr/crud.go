package gofyr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// BulkDeleteRequest defines the body for /bulk-delete.
type BulkDeleteRequest struct {
	IDs []any `json:"ids" binding:"required"`
}

// resolveModel looks up a model in the specified datasource.
func (a *Admin) resolveModel(datasourceName, modelName string) (*Datasource, *ModelInfo, error) {
	ds, ok := a.datasources[datasourceName]
	if !ok {
		return nil, nil, fmt.Errorf("datasource '%s' not found", datasourceName)
	}

	ds.mu.RLock()
	defer ds.mu.RUnlock()

	// Direct match by model Name
	if m, ok := ds.modelsMap[strings.ToLower(modelName)]; ok {
		return ds, m, nil
	}

	// Match by TableName
	for _, m := range ds.Models {
		if strings.EqualFold(m.TableName, modelName) || strings.EqualFold(m.Name, modelName) {
			return ds, m, nil
		}
	}

	return ds, nil, fmt.Errorf("model '%s' not found in datasource '%s'", modelName, datasourceName)
}

// handleListRecords handles GET <BasePath>/api/ds/:datasource/records/:model
func (a *Admin) handleListRecords(c *gin.Context) {
	dsName := c.Param("datasource")
	modelName := c.Param("model")

	ds, model, err := a.resolveModel(dsName, modelName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := ds.DB.Table(model.TableName)

	// Filter out soft-deleted records if deleted_at column exists
	for _, f := range model.Fields {
		if f.DBName == "deleted_at" {
			query = query.Where("deleted_at IS NULL")
			break
		}
	}

	// Search filter
	search := strings.TrimSpace(c.Query("search"))
	if search != "" {
		var searchConditions []string
		var searchArgs []any
		searchTerm := "%" + search + "%"

		for _, f := range model.Fields {
			if f.Searchable || f.GofyrType == "text" || f.Type == "string" {
				searchConditions = append(searchConditions, fmt.Sprintf("%s LIKE ?", f.DBName))
				searchArgs = append(searchArgs, searchTerm)
			}
		}

		if len(searchConditions) > 0 {
			query = query.Where(strings.Join(searchConditions, " OR "), searchArgs...)
		}
	}

	// Column filters: filter[column]=val or query params
	for key, values := range c.Request.URL.Query() {
		if len(values) == 0 || values[0] == "" {
			continue
		}
		val := values[0]
		var colName string
		if strings.HasPrefix(key, "filter[") && strings.HasSuffix(key, "]") {
			colName = strings.TrimSuffix(strings.TrimPrefix(key, "filter["), "]")
		} else {
			// Check if key directly matches a field DBName
			for _, f := range model.Fields {
				if f.DBName == key {
					colName = key
					break
				}
			}
		}

		if colName != "" && isValidColumn(colName, model) {
			query = query.Where(fmt.Sprintf("%s = ?", colName), val)
		}
	}

	// Sorting
	sortParam := c.Query("sort")
	if sortParam != "" {
		parts := strings.Fields(sortParam)
		col := parts[0]
		dir := "asc"
		if len(parts) > 1 && strings.ToLower(parts[1]) == "desc" {
			dir = "desc"
		}
		if isValidColumn(col, model) {
			query = query.Order(fmt.Sprintf("%s %s", col, dir))
		}
	} else {
		// Default sort by primary key descending
		query = query.Order(fmt.Sprintf("%s desc", model.PrimaryKey))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count records: " + err.Error()})
		return
	}

	var records []map[string]any
	if err := query.Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query records: " + err.Error()})
		return
	}

	// Normalize JSON and binary fields for clean JSON output
	for i := range records {
		records[i] = sanitizeRecord(records[i], model)
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, gin.H{
		"data":        records,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

// handleGetRecord handles GET <BasePath>/api/ds/:datasource/records/:model/:id
func (a *Admin) handleGetRecord(c *gin.Context) {
	dsName := c.Param("datasource")
	modelName := c.Param("model")
	id := c.Param("id")

	ds, model, err := a.resolveModel(dsName, modelName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var record map[string]any
	if err := ds.DB.Table(model.TableName).Where(fmt.Sprintf("%s = ?", model.PrimaryKey), id).Take(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sanitizeRecord(record, model)})
}

// handleCreateRecord handles POST <BasePath>/api/ds/:datasource/records/:model
func (a *Admin) handleCreateRecord(c *gin.Context) {
	dsName := c.Param("datasource")
	modelName := c.Param("model")

	ds, model, err := a.resolveModel(dsName, modelName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Prepare data map adhering to model fields
	data := make(map[string]any)
	for _, f := range model.Fields {
		if f.IsPrimaryKey && (payload[f.DBName] == nil && payload[f.Name] == nil) {
			continue // skip primary key if not provided (auto-increment)
		}
		if f.Readonly && !f.IsPrimaryKey {
			continue
		}

		val, exists := payload[f.DBName]
		if !exists {
			val, exists = payload[f.Name]
		}
		if exists {
			// Convert complex types if gallery or json
			if (f.GofyrType == "gallery" || f.GofyrType == "json") && val != nil {
				if _, ok := val.(string); !ok {
					bytes, _ := json.Marshal(val)
					val = string(bytes)
				}
			}
			data[f.DBName] = val
		}
	}

	if err := ds.DB.Table(model.TableName).Create(&data).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create record: " + err.Error()})
		return
	}

	createdID := fmt.Sprintf("%v", data[model.PrimaryKey])
	user := a.CurrentUser(c)
	userEmail := "system"
	var userID uint
	if user != nil {
		userID = user.UserID
		userEmail = user.Email
	}
	a.LogAudit(userID, userEmail, "crud_create", dsName, model.Name, createdID, c.ClientIP(), fmt.Sprintf("Created %s with ID %s", model.Name, createdID))

	c.JSON(http.StatusCreated, gin.H{"data": sanitizeRecord(data, model)})
}

// handleUpdateRecord handles PUT <BasePath>/api/ds/:datasource/records/:model/:id
func (a *Admin) handleUpdateRecord(c *gin.Context) {
	dsName := c.Param("datasource")
	modelName := c.Param("model")
	id := c.Param("id")

	ds, model, err := a.resolveModel(dsName, modelName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Clean updates map
	updates := make(map[string]any)
	for _, f := range model.Fields {
		if f.IsPrimaryKey || f.Readonly {
			continue
		}
		val, exists := payload[f.DBName]
		if !exists {
			val, exists = payload[f.Name]
		}
		if exists {
			if (f.GofyrType == "gallery" || f.GofyrType == "json") && val != nil {
				if _, ok := val.(string); !ok {
					bytes, _ := json.Marshal(val)
					val = string(bytes)
				}
			}
			updates[f.DBName] = val
		}
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No modifiable fields supplied"})
		return
	}

	res := ds.DB.Table(model.TableName).Where(fmt.Sprintf("%s = ?", model.PrimaryKey), id).Updates(updates)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update record: " + res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	user := a.CurrentUser(c)
	userEmail := "system"
	var userID uint
	if user != nil {
		userID = user.UserID
		userEmail = user.Email
	}
	a.LogAudit(userID, userEmail, "crud_update", dsName, model.Name, id, c.ClientIP(), fmt.Sprintf("Updated %s ID %s", model.Name, id))

	// Fetch updated record
	var updatedRecord map[string]any
	_ = ds.DB.Table(model.TableName).Where(fmt.Sprintf("%s = ?", model.PrimaryKey), id).Take(&updatedRecord)

	c.JSON(http.StatusOK, gin.H{"data": sanitizeRecord(updatedRecord, model)})
}

// handleDeleteRecord handles DELETE <BasePath>/api/ds/:datasource/records/:model/:id
func (a *Admin) handleDeleteRecord(c *gin.Context) {
	dsName := c.Param("datasource")
	modelName := c.Param("model")
	id := c.Param("id")

	ds, model, err := a.resolveModel(dsName, modelName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	res := ds.DB.Table(model.TableName).Where(fmt.Sprintf("%s = ?", model.PrimaryKey), id).Delete(nil)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete record: " + res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	user := a.CurrentUser(c)
	userEmail := "system"
	var userID uint
	if user != nil {
		userID = user.UserID
		userEmail = user.Email
	}
	a.LogAudit(userID, userEmail, "crud_delete", dsName, model.Name, id, c.ClientIP(), fmt.Sprintf("Deleted %s ID %s", model.Name, id))

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// handleBulkDeleteRecords handles POST <BasePath>/api/ds/:datasource/records/:model/bulk-delete
func (a *Admin) handleBulkDeleteRecords(c *gin.Context) {
	dsName := c.Param("datasource")
	modelName := c.Param("model")

	ds, model, err := a.resolveModel(dsName, modelName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var req BulkDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid ids array is required"})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No IDs provided", "count": 0})
		return
	}

	res := ds.DB.Table(model.TableName).Where(fmt.Sprintf("%s IN ?", model.PrimaryKey), req.IDs).Delete(nil)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to bulk delete records: " + res.Error.Error()})
		return
	}

	user := a.CurrentUser(c)
	userEmail := "system"
	var userID uint
	if user != nil {
		userID = user.UserID
		userEmail = user.Email
	}
	a.LogAudit(userID, userEmail, "crud_bulk_delete", dsName, model.Name, "", c.ClientIP(), fmt.Sprintf("Bulk deleted %d %s records", res.RowsAffected, model.Name))

	c.JSON(http.StatusOK, gin.H{
		"message": "Records deleted successfully",
		"count":   res.RowsAffected,
	})
}

// isValidColumn verifies if a column name belongs to the model schema.
func isValidColumn(col string, model *ModelInfo) bool {
	col = strings.ToLower(col)
	for _, f := range model.Fields {
		if strings.ToLower(f.DBName) == col {
			return true
		}
	}
	return false
}

// sanitizeRecord unmarshals JSON strings for gallery/json fields so the frontend gets parsed JSON arrays/objects.
func sanitizeRecord(record map[string]any, model *ModelInfo) map[string]any {
	if record == nil {
		return nil
	}
	for _, f := range model.Fields {
		val, exists := record[f.DBName]
		if !exists || val == nil {
			continue
		}

		if (f.GofyrType == "gallery" || f.GofyrType == "json") {
			if strVal, ok := val.(string); ok && strings.HasPrefix(strings.TrimSpace(strVal), "[") || strings.HasPrefix(strings.TrimSpace(strVal), "{") {
				var parsed any
				if err := json.Unmarshal([]byte(strVal), &parsed); err == nil {
					record[f.DBName] = parsed
				}
			}
		}
	}
	return record
}
