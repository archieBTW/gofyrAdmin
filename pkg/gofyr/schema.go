package gofyr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// ModelInfo holds introspected metadata about a GORM model.
type ModelInfo struct {
	Name        string          `json:"name"`
	TableName   string          `json:"table_name"`
	PrimaryKey  string          `json:"primary_key"`
	Fields      []FieldInfo     `json:"fields"`
	Relations   []RelationInfo  `json:"relations"`
	modelType   reflect.Type    // struct type
}

// FieldInfo represents introspected column and UI rendering instructions.
type FieldInfo struct {
	Name         string `json:"name"`          // Go struct field name
	DBName       string `json:"db_name"`       // SQL column name
	Type         string `json:"type"`          // Go/SQL primitive type
	GofyrType    string `json:"gofyr_type"`    // "text", "number", "boolean", "datetime", "image", "gallery", "file", "json"
	Disk         string `json:"disk,omitempty"`
	Folder       string `json:"folder,omitempty"`
	Readonly     bool   `json:"readonly"`
	Searchable   bool   `json:"searchable"`
	Hidden       bool   `json:"hidden"`
	Required     bool   `json:"required"`
	Label        string `json:"label"`
	IsPrimaryKey bool   `json:"is_primary_key"`
}

// RelationInfo captures GORM associations (HasOne, HasMany, BelongsTo, Many2Many).
type RelationInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "BelongsTo", "HasOne", "HasMany", "Many2Many"
	TargetModel string `json:"target_model"`
	ForeignKey  string `json:"foreign_key"`
}

// SchemaResponse is returned by GET <BasePath>/api/schema.
type SchemaResponse struct {
	Title        string                  `json:"title"`
	BasePath     string                  `json:"base_path"`
	DefaultDisk  string                  `json:"default_disk"`
	Disks        []string                `json:"disks"`
	Datasources  []DatasourceSchema      `json:"datasources"`
}

// DatasourceSchema describes a datasource and its registered models.
type DatasourceSchema struct {
	Name      string      `json:"name"`
	Dialect   string      `json:"dialect"`
	IsPrimary bool        `json:"is_primary"`
	Models    []ModelInfo `json:"models"`
}

var schemaCache sync.Map

// ParseModel inspects a GORM model struct using schema.Parse and reflection.
func ParseModel(db *gorm.DB, model any) (*ModelInfo, error) {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("gofyr: model must be a struct or struct pointer, got %T", model)
	}

	modelType := val.Type()
	gormSchema, err := schema.Parse(model, &schemaCache, db.NamingStrategy)
	if err != nil {
		return nil, fmt.Errorf("gofyr: parse gorm schema for %s: %w", modelType.Name(), err)
	}

	info := &ModelInfo{
		Name:       modelType.Name(),
		TableName:  gormSchema.Table,
		PrimaryKey: "id",
		modelType:  modelType,
	}

	if gormSchema.PrioritizedPrimaryField != nil {
		info.PrimaryKey = gormSchema.PrioritizedPrimaryField.DBName
	}

	for _, field := range gormSchema.Fields {
		// Skip ignored GORM fields
		if field.IgnoreMigration {
			continue
		}

		fieldInfo := FieldInfo{
			Name:         field.Name,
			DBName:       field.DBName,
			Type:         field.FieldType.String(),
			Label:        formatLabel(field.Name),
			IsPrimaryKey: field.PrimaryKey,
		}

		// Detect base gofyr_type from Go field type
		fieldInfo.GofyrType = detectBaseType(field.FieldType)

		// Parse `gofyr:"..."` struct tag
		if tagVal, ok := field.Tag.Lookup("gofyr"); ok {
			parseGofyrTag(tagVal, &fieldInfo)
		}

		// If gorm tag specifies not null, mark required
		if gormTag, ok := field.Tag.Lookup("gorm"); ok {
			if strings.Contains(strings.ToLower(gormTag), "not null") && !field.PrimaryKey {
				fieldInfo.Required = true
			}
		}

		info.Fields = append(info.Fields, fieldInfo)
	}

	// Introspect relationships
	for _, rel := range gormSchema.Relationships.Relations {
		relInfo := RelationInfo{
			Name:        rel.Name,
			Type:        string(rel.Type),
			TargetModel: rel.FieldSchema.ModelType.Name(),
		}
		if len(rel.References) > 0 {
			relInfo.ForeignKey = rel.References[0].ForeignKey.DBName
		}
		info.Relations = append(info.Relations, relInfo)
	}

	return info, nil
}

// detectBaseType maps Go reflect types to frontend presentation types.
func detectBaseType(t reflect.Type) string {
	if t == reflect.TypeOf(time.Time{}) || t == reflect.TypeOf(&time.Time{}) {
		return "datetime"
	}

	kind := t.Kind()
	if kind == reflect.Pointer {
		kind = t.Elem().Kind()
	}

	switch kind {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "json"
	case reflect.Map, reflect.Struct:
		return "json"
	default:
		return "text"
	}
}

// parseGofyrTag parses tags like `gofyr:"type:image,disk:s3,folder:avatars,readonly,searchable"`
func parseGofyrTag(tag string, info *FieldInfo) {
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, ":", 2)
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := ""
		if len(kv) == 2 {
			val = strings.TrimSpace(kv[1])
		}

		switch key {
		case "type":
			if val != "" {
				info.GofyrType = val // e.g. "image", "gallery", "file", "textarea", "richtext", "password"
			}
		case "disk":
			info.Disk = val // e.g. "s3", "local"
		case "folder":
			info.Folder = val // e.g. "avatars", "products"
		case "readonly":
			info.Readonly = true
		case "searchable":
			info.Searchable = true
		case "hidden":
			info.Hidden = true
		case "required":
			info.Required = true
		case "label":
			if val != "" {
				info.Label = val
			}
		}
	}
}

// formatLabel formats CamelCase into Title Case label
func formatLabel(s string) string {
	if s == "ID" {
		return "ID"
	}
	var out []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' && (s[i-1] < 'A' || s[i-1] > 'Z') {
			out = append(out, ' ')
		}
		out = append(out, r)
	}
	return string(out)
}

// NewInstance creates a new pointer to an instance of the model.
func (m *ModelInfo) NewInstance() any {
	return reflect.New(m.modelType).Interface()
}

// NewSlice creates a new pointer to a slice of instances of the model.
func (m *ModelInfo) NewSlice() any {
	sliceType := reflect.SliceOf(m.modelType)
	return reflect.New(sliceType).Interface()
}

// CreateCustomModelRequest defines payload from the UI for dynamic model creation.
type CreateCustomModelRequest struct {
	Datasource string      `json:"datasource" binding:"required"`
	Name       string      `json:"name" binding:"required"`
	TableName  string      `json:"table_name" binding:"required"`
	Fields     []FieldInfo `json:"fields" binding:"required"`
}

// handleCreateCustomModel creates a new database table and registers the model in real-time.
func (a *Admin) handleCreateCustomModel(c *gin.Context) {
	var req CreateCustomModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	a.mu.RLock()
	ds, ok := a.datasources[req.Datasource]
	a.mu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Datasource '%s' not found", req.Datasource)})
		return
	}

	cleanTable := sanitizeSQLIdentifier(req.TableName)
	cleanName := strings.TrimSpace(req.Name)
	if cleanTable == "" || cleanName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid name and table_name are required"})
		return
	}

	// Build CREATE TABLE SQL statement
	var colDefs []string
	switch ds.Dialect {
	case "postgres":
		colDefs = append(colDefs, "id SERIAL PRIMARY KEY")
	case "mysql":
		colDefs = append(colDefs, "id BIGINT AUTO_INCREMENT PRIMARY KEY")
	default:
		colDefs = append(colDefs, "id INTEGER PRIMARY KEY AUTOINCREMENT")
	}

	hasCreatedAt := false
	hasUpdatedAt := false

	modelFields := []FieldInfo{
		{
			Name:         "ID",
			DBName:       "id",
			Type:         "uint",
			GofyrType:    "number",
			Label:        "ID",
			IsPrimaryKey: true,
			Readonly:     true,
		},
	}

	for _, f := range req.Fields {
		colName := sanitizeSQLIdentifier(f.DBName)
		if colName == "" || colName == "id" {
			continue
		}
		if colName == "created_at" {
			hasCreatedAt = true
		}
		if colName == "updated_at" {
			hasUpdatedAt = true
		}

		sqlType := "VARCHAR(255)"
		switch f.GofyrType {
		case "number":
			sqlType = "NUMERIC"
		case "boolean":
			sqlType = "BOOLEAN"
		case "datetime":
			sqlType = "TIMESTAMP"
		case "textarea", "json", "gallery":
			sqlType = "TEXT"
		case "image", "file":
			sqlType = "VARCHAR(1024)"
		}

		colDef := fmt.Sprintf("%s %s", colName, sqlType)
		if f.Required {
			colDef += " NOT NULL"
		}
		colDefs = append(colDefs, colDef)

		f.DBName = colName
		if f.Name == "" {
			f.Name = formatLabel(colName)
		}
		if f.Label == "" {
			f.Label = formatLabel(colName)
		}
		modelFields = append(modelFields, f)
	}

	if !hasCreatedAt {
		colDefs = append(colDefs, "created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP")
		modelFields = append(modelFields, FieldInfo{
			Name:      "CreatedAt",
			DBName:    "created_at",
			Type:      "time.Time",
			GofyrType: "datetime",
			Label:     "Created At",
			Readonly:  true,
		})
	}
	if !hasUpdatedAt {
		colDefs = append(colDefs, "updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP")
	}

	createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", cleanTable, strings.Join(colDefs, ", "))
	if err := ds.DB.Exec(createSQL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create database table: " + err.Error()})
		return
	}

	modelInfo := &ModelInfo{
		Name:       cleanName,
		TableName:  cleanTable,
		PrimaryKey: "id",
		Fields:     modelFields,
	}

	ds.mu.Lock()
	ds.Models = append(ds.Models, modelInfo)
	ds.modelsMap[strings.ToLower(modelInfo.Name)] = modelInfo
	ds.modelsMap[strings.ToLower(modelInfo.TableName)] = modelInfo
	ds.mu.Unlock()

	// Persist to gofyr_custom_models on primary DB
	fieldsBytes, _ := json.Marshal(modelFields)
	primaryDB := a.PrimaryDB()
	if primaryDB != nil {
		customRec := GofyrCustomModel{
			Datasource: req.Datasource,
			Name:       cleanName,
			ModelTable: cleanTable,
			FieldsJSON: string(fieldsBytes),
			CreatedAt:  time.Now(),
		}
		primaryDB.Create(&customRec)
	}

	user := a.CurrentUser(c)
	userEmail := "system"
	var userID uint
	if user != nil {
		userID = user.UserID
		userEmail = user.Email
	}
	a.LogAudit(userID, userEmail, "model_create", req.Datasource, cleanName, "", c.ClientIP(), fmt.Sprintf("Created dynamic model %s (%s)", cleanName, cleanTable))

	c.JSON(http.StatusCreated, gin.H{
		"message": "Model created successfully",
		"model":   modelInfo,
	})
}

// LoadCustomModels loads any dynamically defined models from gofyr_custom_models table.
func (a *Admin) LoadCustomModels() error {
	primaryDB := a.PrimaryDB()
	if primaryDB == nil {
		return nil
	}

	var customModels []GofyrCustomModel
	if err := primaryDB.Find(&customModels).Error; err != nil {
		return err
	}

	for _, cm := range customModels {
		a.mu.RLock()
		ds, ok := a.datasources[cm.Datasource]
		a.mu.RUnlock()
		if !ok {
			continue
		}

		var fields []FieldInfo
		_ = json.Unmarshal([]byte(cm.FieldsJSON), &fields)

		modelInfo := &ModelInfo{
			Name:       cm.Name,
			TableName:  cm.ModelTable,
			PrimaryKey: "id",
			Fields:     fields,
		}

		ds.mu.Lock()
		ds.Models = append(ds.Models, modelInfo)
		ds.modelsMap[strings.ToLower(modelInfo.Name)] = modelInfo
		ds.modelsMap[strings.ToLower(modelInfo.TableName)] = modelInfo
		ds.mu.Unlock()
	}

	return nil
}

func sanitizeSQLIdentifier(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var out []rune
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			out = append(out, r)
		}
	}
	return string(out)
}
