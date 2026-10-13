package gofyr

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Datasource wraps a connected *gorm.DB with metadata and registered models.
type Datasource struct {
	Name      string        `json:"name"`
	Dialect   string        `json:"dialect"`
	IsPrimary bool          `json:"is_primary"`
	DB        *gorm.DB      `json:"-"`
	Models    []*ModelInfo  `json:"models"`
	modelsMap map[string]*ModelInfo
	mu        sync.RWMutex
}

// OpenDatasource is a convenience helper connecting to Postgres, MySQL, or SQLite.
func OpenDatasource(dialect, dsn string, gormConfig ...*gorm.Config) (*gorm.DB, error) {
	var cfg *gorm.Config
	if len(gormConfig) > 0 && gormConfig[0] != nil {
		cfg = gormConfig[0]
	} else {
		cfg = &gorm.Config{}
	}

	normalized := strings.ToLower(strings.TrimSpace(dialect))
	switch normalized {
	case "sqlite", "sqlite3":
		return gorm.Open(sqlite.Open(dsn), cfg)
	case "postgres", "postgresql", "pg":
		return gorm.Open(postgres.Open(dsn), cfg)
	case "mysql", "mariadb":
		return gorm.Open(mysql.Open(dsn), cfg)
	default:
		return nil, fmt.Errorf("gofyr: unsupported SQL dialect '%s'. Supported: sqlite, postgres, mysql", dialect)
	}
}

// DetectDialect inspects the GORM dialector name.
func DetectDialect(db *gorm.DB) string {
	if db == nil || db.Dialector == nil {
		return "unknown"
	}
	name := strings.ToLower(db.Dialector.Name())
	switch {
	case strings.Contains(name, "sqlite"):
		return "sqlite"
	case strings.Contains(name, "postgres") || strings.Contains(name, "pg"):
		return "postgres"
	case strings.Contains(name, "mysql") || strings.Contains(name, "mariadb"):
		return "mysql"
	default:
		return name
	}
}

// PingStatus holds connectivity details for a datasource.
type PingStatus struct {
	Name      string        `json:"name"`
	Dialect   string        `json:"dialect"`
	IsPrimary bool          `json:"is_primary"`
	Online    bool          `json:"online"`
	Latency   time.Duration `json:"latency"`
	LatencyMS float64       `json:"latency_ms"`
	Version   string        `json:"version"`
	Error     string        `json:"error,omitempty"`
}

// Ping tests database connectivity and queries the database engine version.
func (ds *Datasource) Ping() PingStatus {
	status := PingStatus{
		Name:      ds.Name,
		Dialect:   ds.Dialect,
		IsPrimary: ds.IsPrimary,
	}

	if ds.DB == nil {
		status.Online = false
		status.Error = "nil database connection"
		return status
	}

	sqlDB, err := ds.DB.DB()
	if err != nil {
		status.Online = false
		status.Error = err.Error()
		return status
	}

	start := time.Now()
	if err := sqlDB.Ping(); err != nil {
		status.Online = false
		status.Latency = time.Since(start)
		status.LatencyMS = float64(status.Latency.Microseconds()) / 1000.0
		status.Error = err.Error()
		return status
	}
	status.Latency = time.Since(start)
	status.LatencyMS = float64(status.Latency.Microseconds()) / 1000.0
	status.Online = true

	// Query version based on dialect
	status.Version = queryDatabaseVersion(sqlDB, ds.Dialect)
	return status
}

func queryDatabaseVersion(db *sql.DB, dialect string) string {
	var query string
	switch dialect {
	case "sqlite":
		query = "SELECT sqlite_version()"
	case "postgres":
		query = "SELECT version()"
	case "mysql":
		query = "SELECT VERSION()"
	default:
		return "Connected"
	}

	var version string
	if err := db.QueryRow(query).Scan(&version); err != nil {
		return "Connected"
	}
	return strings.TrimSpace(version)
}
