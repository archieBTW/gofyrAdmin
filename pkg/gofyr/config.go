package gofyr

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/archieBTW/gofyr/pkg/gofyr/storage"
)

// Config represents configuration options for gofyrAdmin.
type Config struct {
	// BasePath is the HTTP route prefix where gofyrAdmin is mounted (default: "/admin").
	BasePath string `yaml:"base_path" json:"base_path"`

	// Title is the branded display title in the UI (default: "gofyrAdmin").
	Title string `yaml:"title" json:"title"`

	// JWTSecret is the HMAC secret used for signing session and pre-auth tokens.
	JWTSecret string `yaml:"jwt_secret" json:"jwt_secret"`

	// SessionExpiry is the duration for which session tokens remain valid (default: 24h).
	SessionExpiry time.Duration `yaml:"session_expiry" json:"session_expiry"`

	// PreAuthExpiry is the duration for pre-auth tokens before MFA verification (default: 5m).
	PreAuthExpiry time.Duration `yaml:"pre_auth_expiry" json:"pre_auth_expiry"`

	// AuthRequired enables/disables JWT + MFA authentication protection (default: true).
	AuthRequired bool `yaml:"auth_required" json:"auth_required"`

	// AuditLogging controls whether CRUD actions and logins are logged (default: true).
	AuditLogging bool `yaml:"audit_logging" json:"audit_logging"`

	// DefaultDisk is the default storage driver key for uploads (default: "local").
	DefaultDisk string `yaml:"default_disk" json:"default_disk"`

	// LocalStorage configures the local file storage driver.
	LocalStorage storage.LocalConfig `yaml:"local_storage" json:"local_storage"`

	// S3Storage configures the AWS/S3-compatible storage driver.
	S3Storage storage.S3Config `yaml:"s3_storage" json:"s3_storage"`
}

// DefaultConfig returns a Config initialized with sensible production defaults.
func DefaultConfig() Config {
	return Config{
		BasePath:      "/admin",
		Title:         "gofyrAdmin",
		JWTSecret:     generateRandomSecret(32),
		SessionExpiry: 24 * time.Hour,
		PreAuthExpiry: 5 * time.Minute,
		AuthRequired:  true,
		AuditLogging:  true,
		DefaultDisk:   "local",
		LocalStorage: storage.LocalConfig{
			BaseDir:         "./uploads",
			PublicURLPrefix: "/admin/uploads",
		},
	}
}

// Normalize ensures paths have leading slashes and no trailing slashes.
func (c *Config) Normalize() {
	if c.BasePath == "" {
		c.BasePath = "/admin"
	}
	if !strings.HasPrefix(c.BasePath, "/") {
		c.BasePath = "/" + c.BasePath
	}
	c.BasePath = strings.TrimSuffix(c.BasePath, "/")

	if c.Title == "" {
		c.Title = "gofyrAdmin"
	}
	if c.JWTSecret == "" {
		c.JWTSecret = generateRandomSecret(32)
	}
	if c.SessionExpiry == 0 {
		c.SessionExpiry = 24 * time.Hour
	}
	if c.PreAuthExpiry == 0 {
		c.PreAuthExpiry = 5 * time.Minute
	}
	if c.DefaultDisk == "" {
		c.DefaultDisk = "local"
	}
	if c.LocalStorage.BaseDir == "" {
		c.LocalStorage.BaseDir = "./uploads"
	}
	if c.LocalStorage.PublicURLPrefix == "" {
		c.LocalStorage.PublicURLPrefix = c.BasePath + "/uploads"
	}
}

func generateRandomSecret(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "gofyr_admin_secure_secret_fallback_2026"
	}
	return hex.EncodeToString(bytes)
}
