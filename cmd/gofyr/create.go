package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	createFlagDB      string
	createFlagStorage string
	createFlagModule  string
	createFlagSeed    bool
)

var createCmd = &cobra.Command{
	Use:   "create <project-name>",
	Short: "Bootstrap a complete, runnable Go + Gin + GORM project pre-configured with gofyrAdmin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		projectDir, err := filepath.Abs(projectName)
		if err != nil {
			return err
		}

		if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
			return fmt.Errorf("directory '%s' already exists", projectName)
		}

		reader := bufio.NewReader(os.Stdin)

		// 1. Interactive Prompts if flags not provided or default
		dbDriver := strings.ToLower(strings.TrimSpace(createFlagDB))
		if !cmd.Flags().Changed("db") {
			fmt.Println("\nChoose database driver:")
			fmt.Println("  [1] SQLite (Default - Zero Config, Pure Go)")
			fmt.Println("  [2] PostgreSQL")
			fmt.Println("  [3] MySQL")
			fmt.Print("Selection [1]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			switch input {
			case "2", "postgres", "postgresql":
				dbDriver = "postgres"
			case "3", "mysql":
				dbDriver = "mysql"
			default:
				dbDriver = "sqlite"
			}
		}

		storageDriver := strings.ToLower(strings.TrimSpace(createFlagStorage))
		if !cmd.Flags().Changed("storage") {
			fmt.Println("\nChoose media storage backend:")
			fmt.Println("  [1] Local Disk (Default)")
			fmt.Println("  [2] S3 / MinIO (Object Storage)")
			fmt.Print("Selection [1]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			switch input {
			case "2", "s3", "minio":
				storageDriver = "s3"
			default:
				storageDriver = "local"
			}
		}

		moduleName := createFlagModule
		if moduleName == "" {
			moduleName = filepath.Base(projectName)
		}

		fmt.Printf("\nScaffolding new gofyr project '%s'...\n", projectName)
		fmt.Printf("• Database: %s\n• Storage:  %s\n• Module:   %s\n\n", dbDriver, storageDriver, moduleName)

		// 2. Create directory layout
		if err := os.MkdirAll(filepath.Join(projectDir, "internal", "models"), 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(projectDir, "internal", "seed"), 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(projectDir, "uploads"), 0755); err != nil {
			return err
		}

		// 3. Generate go.mod
		goModContent := generateGoMod(moduleName, dbDriver)
		if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			return err
		}

		// 4. Generate internal/models/models.go
		modelsContent := generateModelsFile()
		if err := os.WriteFile(filepath.Join(projectDir, "internal", "models", "models.go"), []byte(modelsContent), 0644); err != nil {
			return err
		}

		// 5. Generate internal/seed/seed.go
		seedContent := generateSeedFile(moduleName)
		if err := os.WriteFile(filepath.Join(projectDir, "internal", "seed", "seed.go"), []byte(seedContent), 0644); err != nil {
			return err
		}

		// 6. Generate main.go
		mainContent := generateMainFile(moduleName, dbDriver, storageDriver, createFlagSeed)
		if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(mainContent), 0644); err != nil {
			return err
		}

		// 7. Generate .env & gofyr.yaml
		envContent, yamlContent := generateConfigFiles(projectName, dbDriver, storageDriver)
		if err := os.WriteFile(filepath.Join(projectDir, ".env"), []byte(envContent), 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(projectDir, "gofyr.yaml"), []byte(yamlContent), 0644); err != nil {
			return err
		}

		// 8. Generate docker-compose.yml if Postgres, MySQL, or S3
		if dbDriver != "sqlite" || storageDriver == "s3" {
			dockerComposeContent := generateDockerCompose(projectName, dbDriver, storageDriver)
			if err := os.WriteFile(filepath.Join(projectDir, "docker-compose.yml"), []byte(dockerComposeContent), 0644); err != nil {
				return err
			}
		}

		// 9. Generate Makefile
		makefileContent := generateMakefile(dbDriver, storageDriver)
		if err := os.WriteFile(filepath.Join(projectDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
			return err
		}

		// 10. Run `go mod tidy` in new directory
		fmt.Println("Running 'go mod tidy' to resolve dependencies...")
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = projectDir
		tidyCmd.Stdout = os.Stdout
		tidyCmd.Stderr = os.Stderr
		_ = tidyCmd.Run()

		// 11. Print stylized welcome banner
		printWelcomeBanner(projectName, dbDriver, storageDriver)

		return nil
	},
}

func init() {
	createCmd.Flags().StringVar(&createFlagDB, "db", "sqlite", "Database driver (sqlite, postgres, mysql)")
	createCmd.Flags().StringVar(&createFlagStorage, "storage", "local", "Media storage backend (local, s3)")
	createCmd.Flags().StringVar(&createFlagModule, "module", "", "Go module path (defaults to <project-name>)")
	createCmd.Flags().BoolVar(&createFlagSeed, "seed", true, "Auto-seed mock models and dev admin user on first boot")

	rootCmd.AddCommand(createCmd)
}

func generateGoMod(moduleName, dbDriver string) string {
	cwd, _ := os.Getwd()
	gofyrPath := cwd
	// If current directory is not repo root, find repo root
	if _, err := os.Stat(filepath.Join(cwd, "pkg", "gofyr")); err != nil {
		gofyrPath = "/Users/torrey/Code/gofyr_admin"
	}

	var dbDep string
	switch dbDriver {
	case "postgres":
		dbDep = "gorm.io/driver/postgres v1.6.3"
	case "mysql":
		dbDep = "gorm.io/driver/mysql v1.6.0"
	default:
		dbDep = "github.com/glebarez/sqlite v1.11.0"
	}

	return fmt.Sprintf(`module %s

go 1.24

require (
	github.com/archieBTW/gofyr v0.0.0
	github.com/gin-gonic/gin v1.12.0
	%s
	gorm.io/gorm v1.31.2
)

replace github.com/archieBTW/gofyr => %s
`, moduleName, dbDep, gofyrPath)
}

func generateModelsFile() string {
	return `package models

import (
	"time"
)

// Customer represents a customer account with avatar photo upload.
type Customer struct {
	ID        uint      ` + "`json:\"id\" gorm:\"primaryKey\"`" + `
	Name      string    ` + "`json:\"name\" gorm:\"size:100;not null\" gofyr:\"searchable,label:Full Name\"`" + `
	Email     string    ` + "`json:\"email\" gorm:\"size:255;uniqueIndex;not null\" gofyr:\"searchable,label:Email Address\"`" + `
	Phone     string    ` + "`json:\"phone\" gorm:\"size:50\" gofyr:\"label:Phone Number\"`" + `
	Avatar    string    ` + "`json:\"avatar\" gorm:\"size:1024\" gofyr:\"type:image,disk:local,folder:customers,label:Profile Photo\"`" + `
	Orders    []Order   ` + "`json:\"orders,omitempty\" gorm:\"foreignKey:CustomerID\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\" gofyr:\"readonly,label:Registered At\"`" + `
}

// Product represents catalog inventory with single photo and gallery arrays.
type Product struct {
	ID          uint      ` + "`json:\"id\" gorm:\"primaryKey\"`" + `
	Title       string    ` + "`json:\"title\" gorm:\"size:255;not null\" gofyr:\"searchable,label:Product Title\"`" + `
	SKU         string    ` + "`json:\"sku\" gorm:\"size:100;uniqueIndex;not null\" gofyr:\"searchable,label:SKU Code\"`" + `
	Price       float64   ` + "`json:\"price\" gorm:\"not null\" gofyr:\"label:Retail Price ($)\"`" + `
	Stock       int       ` + "`json:\"stock\" gorm:\"default:0\" gofyr:\"label:Stock Level\"`" + `
	Image       string    ` + "`json:\"image\" gorm:\"size:1024\" gofyr:\"type:image,disk:local,folder:products,label:Cover Image\"`" + `
	Gallery     string    ` + "`json:\"gallery\" gorm:\"type:text\" gofyr:\"type:gallery,disk:local,folder:products,label:Photo Gallery\"`" + `
	Description string    ` + "`json:\"description\" gorm:\"type:text\" gofyr:\"type:textarea,label:Product Description\"`" + `
	IsActive    bool      ` + "`json:\"is_active\" gorm:\"default:true\" gofyr:\"label:Active Status\"`" + `
	CreatedAt   time.Time ` + "`json:\"created_at\" gofyr:\"readonly,label:Added Date\"`" + `
}

// Order represents customer orders demonstrating relationship and readonly tags.
type Order struct {
	ID          uint      ` + "`json:\"id\" gorm:\"primaryKey\"`" + `
	CustomerID  uint      ` + "`json:\"customer_id\" gorm:\"index;not null\" gofyr:\"label:Customer ID\"`" + `
	Customer    Customer  ` + "`json:\"customer,omitempty\" gorm:\"foreignKey:CustomerID\"`" + `
	OrderNumber string    ` + "`json:\"order_number\" gorm:\"size:100;uniqueIndex;not null\" gofyr:\"searchable,readonly,label:Order Number\"`" + `
	TotalAmount float64   ` + "`json:\"total_amount\" gorm:\"not null\" gofyr:\"label:Total Amount ($)\"`" + `
	Status      string    ` + "`json:\"status\" gorm:\"size:50;default:pending\" gofyr:\"label:Fulfillment Status\"`" + `
	CreatedAt   time.Time ` + "`json:\"created_at\" gofyr:\"readonly,label:Order Date\"`" + `
}
`
}

func generateSeedFile(moduleName string) string {
	return fmt.Sprintf(`package seed

import (
	"fmt"
	"time"

	"github.com/archieBTW/gofyr/pkg/gofyr"
	"%s/internal/models"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Run populates 15 realistic sample records and dev superuser with TOTP credentials.
func Run(db *gorm.DB, admin *gofyr.Admin) {
	// 1. Seed Dev Superuser
	pwdHash, _ := gofyr.HashPassword("admin123")
	devAdmin := gofyr.GofyrAdminUser{
		Email:        "admin@gofyr.local",
		PasswordHash: pwdHash,
		Name:         "Dev Superadmin",
		Role:         "superadmin",
		MFASecret:    "JBSWY3DPEHPK3PXP", // Base32 for "Hello!"
		MFAEnabled:   true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	var existingUser gofyr.GofyrAdminUser
	if err := db.Where("email = ?", devAdmin.Email).First(&existingUser).Error; err != nil {
		db.Create(&devAdmin)
		existingUser = devAdmin
	}

	// Always ensure recovery codes for local testing
	var unusedCount int64
	db.Model(&gofyr.GofyrMFARecoveryCode{}).Where("user_id = ? AND used = ?", existingUser.ID, false).Count(&unusedCount)
	if unusedCount == 0 {
		recHash, _ := bcrypt.GenerateFromPassword([]byte("11112222"), bcrypt.DefaultCost)
		db.Create(&gofyr.GofyrMFARecoveryCode{
			UserID:    existingUser.ID,
			CodeHash:  string(recHash),
			Used:      false,
			CreatedAt: time.Now(),
		})
	}

	// 2. Seed Mock Products if empty
	var productCount int64
	db.Model(&models.Product{}).Count(&productCount)
	if productCount == 0 {
		products := []models.Product{
			{
				Title:       "Mechanical Keyboard Pro X",
				SKU:         "KB-MECH-PRO",
				Price:       179.99,
				Stock:       38,
				Image:       "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=800&q=80",
				Gallery:     "[\"https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=800&q=80\"]",
				Description: "Wireless hot-swappable mechanical keyboard with lubricated switches.",
				IsActive:    true,
				CreatedAt:   time.Now().Add(-48 * time.Hour),
			},
			{
				Title:       "Ultrawide Curved Monitor 34\"",
				SKU:         "MON-ULTRA-34",
				Price:       749.00,
				Stock:       14,
				Image:       "https://images.unsplash.com/photo-1527443224154-c4a3942d3acf?w=800&q=80",
				Gallery:     "[\"https://images.unsplash.com/photo-1527443224154-c4a3942d3acf?w=800&q=80\"]",
				Description: "144Hz 1ms IPS gaming and productivity monitor with USB-C 90W charging.",
				IsActive:    true,
				CreatedAt:   time.Now().Add(-36 * time.Hour),
			},
			{
				Title:       "Noise-Canceling Studio Headphones",
				SKU:         "AUD-ANC-STUDIO",
				Price:       329.50,
				Stock:       52,
				Image:       "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=800&q=80",
				Gallery:     "[\"https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=800&q=80\"]",
				Description: "Custom planar drivers delivering audiophile acoustics with 40-hour battery life.",
				IsActive:    true,
				CreatedAt:   time.Now().Add(-24 * time.Hour),
			},
			{
				Title:       "Ergonomic Mesh Task Chair",
				SKU:         "CHR-ERGO-01",
				Price:       489.00,
				Stock:       22,
				Image:       "https://images.unsplash.com/photo-1580481077195-c2f82d2780e5?w=800&q=80",
				Gallery:     "[\"https://images.unsplash.com/photo-1580481077195-c2f82d2780e5?w=800&q=80\"]",
				Description: "Dynamic lumbar support and 4D adjustable armrests.",
				IsActive:    true,
				CreatedAt:   time.Now().Add(-12 * time.Hour),
			},
			{
				Title:       "Aluminum Monitor Desk Mount",
				SKU:         "ACC-DSK-ARM",
				Price:       89.95,
				Stock:       75,
				Image:       "https://images.unsplash.com/photo-1593642632823-8f785ba67e45?w=800&q=80",
				Gallery:     "[\"https://images.unsplash.com/photo-1593642632823-8f785ba67e45?w=800&q=80\"]",
				Description: "Heavy-duty gas spring arm supporting screens up to 40 inches.",
				IsActive:    true,
				CreatedAt:   time.Now().Add(-6 * time.Hour),
			},
		}
		for _, p := range products {
			db.Create(&p)
		}
	}

	// 3. Seed Mock Customers if empty
	var customerCount int64
	db.Model(&models.Customer{}).Count(&customerCount)
	if customerCount == 0 {
		customers := []models.Customer{
			{Name: "Torrey Vance", Email: "torrey@gofyr.local", Phone: "+1-555-0101", Avatar: "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=400&q=80", CreatedAt: time.Now().Add(-72 * time.Hour)},
			{Name: "Jordan Hayes", Email: "jordan@example.com", Phone: "+1-555-0102", Avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=400&q=80", CreatedAt: time.Now().Add(-60 * time.Hour)},
			{Name: "Sophia Lin", Email: "sophia@example.com", Phone: "+1-555-0103", Avatar: "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=400&q=80", CreatedAt: time.Now().Add(-50 * time.Hour)},
			{Name: "Marcus Brody", Email: "marcus@example.com", Phone: "+1-555-0104", Avatar: "https://images.unsplash.com/photo-1500648767791-00dcc994a43e?w=400&q=80", CreatedAt: time.Now().Add(-40 * time.Hour)},
			{Name: "Emma Watson", Email: "emma@example.com", Phone: "+1-555-0105", Avatar: "https://images.unsplash.com/photo-1438761681033-6461ffad8d80?w=400&q=80", CreatedAt: time.Now().Add(-30 * time.Hour)},
		}
		for _, c := range customers {
			db.Create(&c)
		}
	}

	// 4. Seed Mock Orders if empty
	var orderCount int64
	db.Model(&models.Order{}).Count(&orderCount)
	if orderCount == 0 {
		orders := []models.Order{
			{CustomerID: 1, OrderNumber: "ORD-94810", TotalAmount: 928.99, Status: "fulfilled", CreatedAt: time.Now().Add(-20 * time.Hour)},
			{CustomerID: 2, OrderNumber: "ORD-94811", TotalAmount: 329.50, Status: "shipped", CreatedAt: time.Now().Add(-15 * time.Hour)},
			{CustomerID: 3, OrderNumber: "ORD-94812", TotalAmount: 1238.00, Status: "processing", CreatedAt: time.Now().Add(-10 * time.Hour)},
			{CustomerID: 4, OrderNumber: "ORD-94813", TotalAmount: 179.99, Status: "pending", CreatedAt: time.Now().Add(-5 * time.Hour)},
			{CustomerID: 5, OrderNumber: "ORD-94814", TotalAmount: 578.95, Status: "fulfilled", CreatedAt: time.Now().Add(-2 * time.Hour)},
		}
		for _, o := range orders {
			db.Create(&o)
		}
	}

	// Print startup banner with dev credentials and ASCII QR code
	fmt.Println("\n=============================================================")
	fmt.Println("  gofyrAdmin Development Superuser Ready")
	fmt.Println("=============================================================")
	fmt.Println("  Email:            admin@gofyr.local")
	fmt.Println("  Password:         admin123")
	fmt.Println("  Recovery Code:    1111-2222")
	fmt.Println("  TOTP Secret:      JBSWY3DPEHPK3PXP")
	fmt.Println("-------------------------------------------------------------")

	qr, err := qrcode.New("otpauth://totp/gofyrAdmin:admin@gofyr.local?secret=JBSWY3DPEHPK3PXP&issuer=gofyrAdmin", qrcode.Medium)
	if err == nil {
		fmt.Println("  Scan with Authenticator App (Google Authenticator / 1Password):")
		fmt.Println(qr.ToSmallString(false))
	}
	fmt.Println("=============================================================\n")
}
`, moduleName)
}

func generateMainFile(moduleName, dbDriver, storageDriver string, autoSeed bool) string {
	var dbImport, dbConn string
	switch dbDriver {
	case "postgres":
		dbImport = `"gorm.io/driver/postgres"`
		dbConn = `dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=app port=5432 sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})`
	case "mysql":
		dbImport = `"gorm.io/driver/mysql"`
		dbConn = `dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "root:root@tcp(localhost:3306)/app?charset=utf8mb4&parseTime=True&loc=Local"
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})`
	default:
		dbImport = `"github.com/glebarez/sqlite"`
		dbConn = `db, err := gorm.Open(sqlite.Open("./gofyr_dev.db"), &gorm.Config{})`
	}

	var storageConfig string
	if storageDriver == "s3" {
		storageConfig = `cfg.DefaultDisk = "s3"
	cfg.S3Storage = storage.S3Config{
		Bucket:       os.Getenv("S3_BUCKET"),
		Region:       "us-east-1",
		Endpoint:     os.Getenv("S3_ENDPOINT"),
		AccessKey:    os.Getenv("S3_ACCESS_KEY"),
		SecretKey:    os.Getenv("S3_SECRET_KEY"),
		UsePathStyle: true,
	}`
	} else {
		storageConfig = `cfg.DefaultDisk = "local"
	cfg.LocalStorage = storage.LocalConfig{
		BaseDir:         "./uploads",
		PublicURLPrefix: "/gofyr/uploads",
	}`
	}

	return fmt.Sprintf(`package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/archieBTW/gofyr/pkg/gofyr"
	"github.com/archieBTW/gofyr/pkg/gofyr/storage"
	%s
	"%s/internal/models"
	"%s/internal/seed"
	"gorm.io/gorm"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 1. Connect to %s Database
	%s
	if err != nil {
		log.Fatalf("Failed to connect database: %%v", err)
	}

	// 2. Auto-migrate domain models
	if err := db.AutoMigrate(&models.Customer{}, &models.Product{}, &models.Order{}); err != nil {
		log.Fatalf("AutoMigrate failed: %%v", err)
	}

	// 3. Configure gofyrAdmin Drop-In Engine
	cfg := gofyr.DefaultConfig()
	cfg.BasePath = "/gofyr"
	cfg.Title = "%s"
	%s

	admin, err := gofyr.New(cfg)
	if err != nil {
		log.Fatalf("gofyr.New failed: %%v", err)
	}

	// Register Primary Datasource
	if err := admin.AddDatasource("primary", db, true); err != nil {
		log.Fatalf("AddDatasource failed: %%v", err)
	}

	// Register Models with gofyr struct tags
	_ = admin.RegisterModel("primary", &models.Customer{})
	_ = admin.RegisterModel("primary", &models.Product{})
	_ = admin.RegisterModel("primary", &models.Order{})

	// 4. Auto-Seed Development Data
	if os.Getenv("GOFYR_AUTO_SEED") != "false" {
		seed.Run(db, admin)
	}

	// 5. Initialize Gin Engine and Mount gofyrAdmin
	r := gin.Default()

	if err := admin.Mount(r); err != nil {
		log.Fatalf("admin.Mount failed: %%v", err)
	}

	// 6. Host Application REST Routes (Introspected by API Workbench)
	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"version":   "1.0.0",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
		})

		api.GET("/products", func(c *gin.Context) {
			var products []models.Product
			db.Limit(10).Find(&products)
			c.JSON(http.StatusOK, gin.H{
				"products": products,
				"count":    len(products),
			})
		})

		api.POST("/orders", func(c *gin.Context) {
			var req struct {
				CustomerID  uint    `+"`json:\"customer_id\" binding:\"required\"`"+`
				TotalAmount float64 `+"`json:\"total_amount\" binding:\"required\"`"+`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			order := models.Order{
				CustomerID:  req.CustomerID,
				OrderNumber: fmt.Sprintf("ORD-%%d", time.Now().Unix()),
				TotalAmount: req.TotalAmount,
				Status:      "pending",
				CreatedAt:   time.Now(),
			}
			db.Create(&order)
			c.JSON(http.StatusCreated, gin.H{"order": order})
		})
	}

	// Root redirect to /gofyr
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/gofyr")
	})

	fmt.Printf("\n🚀 Server listening on http://localhost:%%s\n", port)
	fmt.Printf("🔥 gofyrAdmin Panel: http://localhost:%%s/gofyr\n\n", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server stopped: %%v", err)
	}
}
`, dbImport, moduleName, moduleName, dbDriver, dbConn, moduleName, storageConfig)
}

func generateConfigFiles(projectName, dbDriver, storageDriver string) (env, yaml string) {
	var dbDSN string
	switch dbDriver {
	case "postgres":
		dbDSN = "host=localhost user=postgres password=postgres dbname=app port=5432 sslmode=disable"
	case "mysql":
		dbDSN = "root:root@tcp(localhost:3306)/app?charset=utf8mb4&parseTime=True&loc=Local"
	default:
		dbDSN = "./gofyr_dev.db"
	}

	env = fmt.Sprintf(`PORT=8080
DATABASE_URL=%s
GOFYR_AUTO_SEED=true
JWT_SECRET=super-secret-dev-key-%s
S3_BUCKET=%s-media
S3_ENDPOINT=http://localhost:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
`, dbDSN, projectName, projectName)

	yaml = fmt.Sprintf(`app:
  title: "%s"
  base_path: "/gofyr"
  jwt_secret: "super-secret-dev-key-%s"
  auth_required: true
  audit_logging: true

datasources:
  - name: "primary"
    dialect: "%s"
    dsn: "%s"
    is_primary: true

storage:
  default_disk: "%s"
  local:
    base_dir: "./uploads"
    public_url_prefix: "/gofyr/uploads"
  s3:
    bucket: "%s-media"
    region: "us-east-1"
    endpoint: "http://localhost:9000"
    access_key: "minioadmin"
    secret_key: "minioadmin"
    use_path_style: true
`, projectName, projectName, dbDriver, dbDSN, storageDriver, projectName)

	return env, yaml
}

func generateDockerCompose(projectName, dbDriver, storageDriver string) string {
	var services []string

	if dbDriver == "postgres" {
		services = append(services, `  postgres:
    image: postgres:16-alpine
    container_name: `+projectName+`-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: app
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data`)
	} else if dbDriver == "mysql" {
		services = append(services, `  mysql:
    image: mysql:8.0
    container_name: `+projectName+`-mysql
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: app
    ports:
      - "3306:3306"
    volumes:
      - mysqldata:/var/lib/mysql`)
	}

	if storageDriver == "s3" {
		services = append(services, `  minio:
    image: minio/minio:latest
    container_name: `+projectName+`-minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - miniodata:/data
  createbuckets:
    image: minio/mc:latest
    depends_on:
      - minio
    entrypoint: >
      /bin/sh -c "
      /usr/bin/mc alias set local http://minio:9000 minioadmin minioadmin;
      /usr/bin/mc mb local/`+projectName+`-media;
      /usr/bin/mc anonymous set public local/`+projectName+`-media;
      exit 0;
      "`)
	}

	var volumes []string
	if dbDriver == "postgres" {
		volumes = append(volumes, "  pgdata:")
	} else if dbDriver == "mysql" {
		volumes = append(volumes, "  mysqldata:")
	}
	if storageDriver == "s3" {
		volumes = append(volumes, "  miniodata:")
	}

	return fmt.Sprintf(`version: '3.8'

services:
%s

volumes:
%s
`, strings.Join(services, "\n\n"), strings.Join(volumes, "\n"))
}

func generateMakefile(dbDriver, storageDriver string) string {
	hasDocker := dbDriver != "sqlite" || storageDriver == "s3"
	dockerTargets := ""
	if hasDocker {
		dockerTargets = `
docker-up:
	docker compose up -d

docker-down:
	docker compose down
`
	}

	return fmt.Sprintf(`.PHONY: run tidy seed%s

run:
	go run .

tidy:
	go mod tidy

seed:
	GOFYR_AUTO_SEED=true go run .
%s`, func() string {
		if hasDocker {
			return " docker-up docker-down"
		}
		return ""
	}(), dockerTargets)
}

func printWelcomeBanner(projectName, dbDriver, storageDriver string) {
	fmt.Println("\n=============================================================")
	fmt.Printf("🎉 Project '%s' bootstrapped successfully!\n", projectName)
	fmt.Println("=============================================================")
	fmt.Println("Next steps:")
	fmt.Printf("  1. cd %s\n", projectName)
	if dbDriver != "sqlite" || storageDriver == "s3" {
		fmt.Println("  2. docker compose up -d    # Start local DB / MinIO")
		fmt.Println("  3. go run .")
	} else {
		fmt.Println("  2. go run .")
	}
	fmt.Println("\nAdmin Console: http://localhost:8080/gofyr")
	fmt.Println("Credentials:   admin@gofyr.local / admin123")
	fmt.Println("=============================================================\n")
}
