package gofyr

import (
	"time"

	"gorm.io/gorm"
)

// GofyrAdminUser represents an administrator authorized to access gofyrAdmin.
type GofyrAdminUser struct {
	ID           uint      `json:"id" gorm:"primarykey"`
	Email        string    `json:"email" gorm:"uniqueIndex;not null;size:255"`
	PasswordHash string    `json:"-" gorm:"not null"`
	Name         string    `json:"name" gorm:"size:255"`
	Role         string    `json:"role" gorm:"size:50;default:admin"`
	MFASecret    string    `json:"-" gorm:"size:255"`
	MFAEnabled   bool      `json:"mfa_enabled" gorm:"default:false"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (GofyrAdminUser) TableName() string {
	return "gofyr_admin_users"
}

// GofyrMFARecoveryCode stores single-use recovery codes for locked-out admins.
type GofyrMFARecoveryCode struct {
	ID        uint       `json:"id" gorm:"primarykey"`
	UserID    uint       `json:"user_id" gorm:"index;not null"`
	CodeHash  string     `json:"-" gorm:"not null;size:255"`
	Used      bool       `json:"used" gorm:"default:false"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func (GofyrMFARecoveryCode) TableName() string {
	return "gofyr_mfa_recovery_codes"
}

// GofyrAuditLog tracks sensitive actions, logins, and model mutations.
type GofyrAuditLog struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	UserID     uint      `json:"user_id" gorm:"index"`
	UserEmail  string    `json:"user_email" gorm:"size:255"`
	Action     string    `json:"action" gorm:"size:100;not null"`
	Datasource string    `json:"datasource" gorm:"size:100"`
	Model      string    `json:"model" gorm:"size:100"`
	RecordID   string    `json:"record_id" gorm:"size:100"`
	IPAddress  string    `json:"ip_address" gorm:"size:50"`
	Details    string    `json:"details" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
}

func (GofyrAuditLog) TableName() string {
	return "gofyr_audit_logs"
}

// GofyrMediaFile catalogs uploaded media files across Local disk and S3.
type GofyrMediaFile struct {
	ID           uint           `json:"id" gorm:"primarykey"`
	Disk         string         `json:"disk" gorm:"size:50;not null"`
	Folder       string         `json:"folder" gorm:"size:255"`
	Filename     string         `json:"filename" gorm:"size:255;not null"`
	OriginalName string         `json:"original_name" gorm:"size:255;not null"`
	MimeType     string         `json:"mime_type" gorm:"size:100;not null"`
	Size         int64          `json:"size"`
	URL          string         `json:"url" gorm:"size:1024;not null"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (GofyrMediaFile) TableName() string {
	return "gofyr_media_files"
}

// GofyrCustomModel persists dynamically created models from the UI.
type GofyrCustomModel struct {
	ID         uint      `json:"id" gorm:"primarykey"`
	Datasource string    `json:"datasource" gorm:"size:100;not null"`
	Name       string    `json:"name" gorm:"size:100;not null"`
	ModelTable string    `json:"table_name" gorm:"column:table_name;size:100;not null"`
	FieldsJSON string    `json:"fields_json" gorm:"type:text;not null"`
	CreatedAt  time.Time `json:"created_at"`
}

func (GofyrCustomModel) TableName() string {
	return "gofyr_custom_models"
}

// GofyrCustomRoute persists custom mock routes created from the UI.
type GofyrCustomRoute struct {
	ID           uint      `json:"id" gorm:"primarykey"`
	Method       string    `json:"method" gorm:"size:20;not null"`
	Path         string    `json:"path" gorm:"size:255;not null"`
	StatusCode   int       `json:"status_code" gorm:"default:200"`
	HeadersJSON  string    `json:"headers_json" gorm:"type:text"`
	ResponseBody string    `json:"response_body" gorm:"type:text"`
	Description  string    `json:"description" gorm:"size:255"`
	CreatedAt    time.Time `json:"created_at"`
}

func (GofyrCustomRoute) TableName() string {
	return "gofyr_custom_routes"
}

// InternalModels returns all gofyr internal GORM models for auto-migration.
func InternalModels() []any {
	return []any{
		&GofyrAdminUser{},
		&GofyrMFARecoveryCode{},
		&GofyrAuditLog{},
		&GofyrMediaFile{},
		&GofyrCustomModel{},
		&GofyrCustomRoute{},
	}
}
