package gofyr

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/archieBTW/gofyr/pkg/gofyr/storage"
	"gorm.io/gorm"
)

// Admin is the central gofyr engine managing datasources, schemas, storage drivers, and routes.
type Admin struct {
	cfg            Config
	datasources    map[string]*Datasource
	primaryDS      string
	storageDrivers map[string]storage.Driver
	hostEngine     *gin.Engine
	targetEngine   *gin.Engine
	mu             sync.RWMutex
}

// New creates and initializes a new gofyrAdmin instance with the provided config.
func New(cfg Config) (*Admin, error) {
	cfg.Normalize()

	admin := &Admin{
		cfg:            cfg,
		datasources:    make(map[string]*Datasource),
		storageDrivers: make(map[string]storage.Driver),
	}

	// Initialize default local storage driver
	localDriver, err := storage.NewLocal(cfg.LocalStorage)
	if err != nil {
		return nil, fmt.Errorf("gofyr: failed to initialize local storage driver: %w", err)
	}
	admin.storageDrivers["local"] = localDriver

	// Initialize S3 storage driver if Bucket is configured
	if cfg.S3Storage.Bucket != "" {
		s3Driver, err := storage.NewS3(context.Background(), cfg.S3Storage)
		if err == nil {
			admin.storageDrivers["s3"] = s3Driver
		}
	}

	return admin, nil
}

// Config returns the current active configuration.
func (a *Admin) Config() Config {
	return a.cfg
}

// PrimaryDB returns the *gorm.DB of the designated primary datasource.
func (a *Admin) PrimaryDB() *gorm.DB {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.primaryDS == "" {
		for _, ds := range a.datasources {
			if ds.IsPrimary {
				return ds.DB
			}
		}
		// Fallback to first datasource
		for _, ds := range a.datasources {
			return ds.DB
		}
		return nil
	}
	if ds, ok := a.datasources[a.primaryDS]; ok {
		return ds.DB
	}
	return nil
}

// AddDatasource registers a *gorm.DB with gofyrAdmin.
// If isPrimary is true (or if it is the first registered datasource),
// internal tables (gofyr_*) are automatically migrated onto it.
func (a *Admin) AddDatasource(name string, db *gorm.DB, isPrimary bool) error {
	if db == nil {
		return fmt.Errorf("gofyr: db cannot be nil for datasource '%s'", name)
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = "primary"
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	dialect := DetectDialect(db)

	ds := &Datasource{
		Name:      name,
		Dialect:   dialect,
		IsPrimary: isPrimary,
		DB:        db,
		Models:    make([]*ModelInfo, 0),
		modelsMap: make(map[string]*ModelInfo),
	}

	if isPrimary || len(a.datasources) == 0 {
		ds.IsPrimary = true
		a.primaryDS = name
	}

	a.datasources[name] = ds

	// If this is the primary datasource, auto-migrate internal gofyr tables
	if ds.IsPrimary {
		if err := db.AutoMigrate(InternalModels()...); err != nil {
			return fmt.Errorf("gofyr: failed to auto-migrate internal tables on '%s': %w", name, err)
		}
	}

	return nil
}

// RegisterModel inspects and registers a GORM model on the given datasource.
func (a *Admin) RegisterModel(datasourceName string, model any) error {
	a.mu.Lock()
	ds, ok := a.datasources[datasourceName]
	a.mu.Unlock()

	if !ok {
		return fmt.Errorf("gofyr: datasource '%s' not registered yet", datasourceName)
	}

	modelInfo, err := ParseModel(ds.DB, model)
	if err != nil {
		return err
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Update existing if already registered
	found := false
	for i, m := range ds.Models {
		if strings.EqualFold(m.Name, modelInfo.Name) {
			ds.Models[i] = modelInfo
			found = true
			break
		}
	}
	if !found {
		ds.Models = append(ds.Models, modelInfo)
	}
	ds.modelsMap[strings.ToLower(modelInfo.Name)] = modelInfo
	ds.modelsMap[strings.ToLower(modelInfo.TableName)] = modelInfo

	return nil
}

// AddStorageDriver adds or overrides a storage driver.
func (a *Admin) AddStorageDriver(name string, driver storage.Driver) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.storageDrivers[name] = driver
}

// GetStorageDriver retrieves a storage driver by name.
func (a *Admin) GetStorageDriver(name string) (storage.Driver, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	d, ok := a.storageDrivers[name]
	return d, ok
}

// SetTargetEngine designates a target API engine for workbench route introspection and execution.
func (a *Admin) SetTargetEngine(target *gin.Engine) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.targetEngine = target
}

