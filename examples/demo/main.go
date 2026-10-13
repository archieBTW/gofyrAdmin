package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/archieBTW/gofyr/pkg/gofyr"
	"github.com/archieBTW/gofyr/pkg/gofyr/storage"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Primary DB Models

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"size:100;not null" gofyr:"searchable,label:Username"`
	Email     string    `json:"email" gorm:"size:255;not null" gofyr:"searchable,label:Email Address"`
	Role      string    `json:"role" gorm:"size:50;default:member" gofyr:"label:Role"`
	Avatar    string    `json:"avatar" gorm:"size:1024" gofyr:"type:image,disk:local,folder:avatars,label:Profile Photo"`
	IsActive  bool      `json:"is_active" gorm:"default:true" gofyr:"label:Active Status"`
	Bio       string    `json:"bio" gorm:"type:text" gofyr:"type:textarea,label:Biography"`
	CreatedAt time.Time `json:"created_at" gofyr:"readonly,label:Joined Date"`
}

type Product struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title" gorm:"size:255;not null" gofyr:"searchable,label:Product Title"`
	SKU         string    `json:"sku" gorm:"size:100;not null" gofyr:"searchable,label:SKU Code"`
	Price       float64   `json:"price" gorm:"not null" gofyr:"label:Retail Price"`
	Stock       int       `json:"stock" gorm:"default:0" gofyr:"label:Inventory Stock"`
	CoverImage  string    `json:"cover_image" gorm:"size:1024" gofyr:"type:image,disk:local,folder:products,label:Cover Image"`
	Gallery     string    `json:"gallery" gorm:"type:text" gofyr:"type:gallery,disk:local,folder:products,label:Product Gallery"`
	Description string    `json:"description" gorm:"type:text" gofyr:"type:textarea,label:Full Description"`
	IsPublished bool      `json:"is_published" gorm:"default:true" gofyr:"label:Published"`
	CreatedAt   time.Time `json:"created_at" gofyr:"readonly,label:Created At"`
}

type Article struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title" gorm:"size:255;not null" gofyr:"searchable,label:Article Headline"`
	Slug        string    `json:"slug" gorm:"size:255;not null" gofyr:"label:URL Slug"`
	Attachment  string    `json:"attachment" gorm:"size:1024" gofyr:"type:file,disk:local,folder:documents,label:PDF Attachment"`
	Views       int       `json:"views" gorm:"default:0" gofyr:"readonly,label:Page Views"`
	PublishedAt time.Time `json:"published_at" gofyr:"label:Publication Date"`
}

// Secondary Analytics DB Models

type EventMetric struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	MetricName string    `json:"metric_name" gorm:"size:100;not null" gofyr:"searchable,label:Metric Name"`
	Category   string    `json:"category" gorm:"size:100" gofyr:"label:Category"`
	Value      float64   `json:"value" gofyr:"label:Recorded Value"`
	RecordedAt time.Time `json:"recorded_at" gofyr:"label:Timestamp"`
}

type VisitorLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	IPAddress  string    `json:"ip_address" gorm:"size:50" gofyr:"searchable,label:IP Address"`
	UserAgent  string    `json:"user_agent" gorm:"size:255" gofyr:"label:User Agent"`
	Path       string    `json:"path" gorm:"size:255" gofyr:"searchable,label:Requested Path"`
	StatusCode int       `json:"status_code" gofyr:"label:HTTP Status"`
	CreatedAt  time.Time `json:"created_at" gofyr:"readonly,label:Recorded At"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 1. Initialize Host Gin Engine
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 2. Open Primary and Secondary Databases
	_ = os.MkdirAll("./examples/demo/data", 0755)
	primaryDBPath := "./examples/demo/data/demo_primary.db"
	analyticsDBPath := "./examples/demo/data/demo_analytics.db"

	primaryDB, err := gorm.Open(sqlite.Open(primaryDBPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to open primary database: %v", err)
	}

	analyticsDB, err := gorm.Open(sqlite.Open(analyticsDBPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to open analytics database: %v", err)
	}

	// Auto-migrate domain models
	_ = primaryDB.AutoMigrate(&User{}, &Product{}, &Article{})
	_ = analyticsDB.AutoMigrate(&EventMetric{}, &VisitorLog{})

	// 3. Configure gofyrAdmin Drop-in Engine
	cfg := gofyr.DefaultConfig()
	cfg.BasePath = "/admin"
	cfg.Title = "gofyrAdmin Demo"
	cfg.LocalStorage.BaseDir = "./examples/demo/uploads"
	cfg.LocalStorage.PublicURLPrefix = "/admin/uploads"

	admin, err := gofyr.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create gofyrAdmin engine: %v", err)
	}

	// Register Primary Datasource (auto-migrates gofyr_* tables)
	if err := admin.AddDatasource("primary", primaryDB, true); err != nil {
		log.Fatalf("Failed to add primary datasource: %v", err)
	}

	// Register Secondary Datasource
	if err := admin.AddDatasource("analytics", analyticsDB, false); err != nil {
		log.Fatalf("Failed to add analytics datasource: %v", err)
	}

	// Register Models with custom gofyr tags
	_ = admin.RegisterModel("primary", &User{})
	_ = admin.RegisterModel("primary", &Product{})
	_ = admin.RegisterModel("primary", &Article{})

	_ = admin.RegisterModel("analytics", &EventMetric{})
	_ = admin.RegisterModel("analytics", &VisitorLog{})

	// Seed Sample Data and Superuser
	seedDemoData(primaryDB, analyticsDB, cfg.LocalStorage)

	// 4. Mount gofyrAdmin Panel onto Gin Engine
	if err := admin.Mount(r); err != nil {
		log.Fatalf("Failed to mount gofyrAdmin: %v", err)
	}

	// 5. Host Application Sample Routes (for API Workbench introspection)
	apiV1 := r.Group("/api/v1")
	{
		apiV1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"version":   "1.0.0",
				"service":   "gofyr-host-app",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
		})

		apiV1.GET("/products", func(c *gin.Context) {
			var products []Product
			primaryDB.Limit(10).Find(&products)
			c.JSON(http.StatusOK, gin.H{
				"products": products,
				"count":    len(products),
			})
		})

		apiV1.GET("/products/:id", func(c *gin.Context) {
			id := c.Param("id")
			var product Product
			if err := primaryDB.First(&product, id).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"product": product})
		})

		apiV1.POST("/checkout/quote", func(c *gin.Context) {
			var body struct {
				Items []struct {
					ProductID uint `json:"product_id"`
					Quantity  int  `json:"quantity"`
				} `json:"items"`
				CouponCode string `json:"coupon_code"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid checkout payload: " + err.Error()})
				return
			}

			subtotal := 0.0
			for _, item := range body.Items {
				var p Product
				if err := primaryDB.First(&p, item.ProductID).Error; err == nil {
					subtotal += p.Price * float64(item.Quantity)
				}
			}

			discount := 0.0
			if strings.ToUpper(body.CouponCode) == "GOFYR20" {
				discount = subtotal * 0.20
			}

			c.JSON(http.StatusOK, gin.H{
				"subtotal":   subtotal,
				"discount":   discount,
				"tax":        (subtotal - discount) * 0.08,
				"total":      (subtotal - discount) * 1.08,
				"items_count": len(body.Items),
			})
		})

		apiV1.GET("/metrics/summary", func(c *gin.Context) {
			var metricCount int64
			var visitorCount int64
			analyticsDB.Model(&EventMetric{}).Count(&metricCount)
			analyticsDB.Model(&VisitorLog{}).Count(&visitorCount)

			c.JSON(http.StatusOK, gin.H{
				"metrics_recorded": metricCount,
				"visitors_logged":  visitorCount,
				"primary_db_users": 5,
			})
		})
	}

	// Root redirect to /admin
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/admin")
	})

	fmt.Println("=============================================================")
	fmt.Printf("🔥 gofyrAdmin Demo Server running on http://127.0.0.1:%s\n", port)
	fmt.Printf("   Admin Panel:       http://127.0.0.1:%s/admin\n", port)
	fmt.Printf("   Credentials:       admin@gofyr.dev / admin123\n")
	fmt.Printf("   Recovery Code:     1111-2222\n")
	fmt.Printf("   TOTP Secret:       JBSWY3DPEHPK3PXP\n")
	fmt.Println("=============================================================")

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}

func seedDemoData(primaryDB, analyticsDB *gorm.DB, localCfg storage.LocalConfig) {
	// Seed Admin User
	pwdHash, _ := gofyr.HashPassword("admin123")
	adminUser := gofyr.GofyrAdminUser{
		Email:        "admin@gofyr.dev",
		PasswordHash: pwdHash,
		Name:         "Torrey (Superadmin)",
		Role:         "superadmin",
		MFASecret:    "JBSWY3DPEHPK3PXP", // Base32 for "Hello!"
		MFAEnabled:   true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	var existingUser gofyr.GofyrAdminUser
	if err := primaryDB.Where("email = ?", adminUser.Email).First(&existingUser).Error; err != nil {
		primaryDB.Create(&adminUser)
		existingUser = adminUser
	}

	// Always ensure at least one active recovery code for demo: 1111-2222
	var unusedCount int64
	primaryDB.Model(&gofyr.GofyrMFARecoveryCode{}).Where("user_id = ? AND used = ?", existingUser.ID, false).Count(&unusedCount)
	if unusedCount == 0 {
		recHash, _ := bcrypt.GenerateFromPassword([]byte("11112222"), bcrypt.DefaultCost)
		primaryDB.Create(&gofyr.GofyrMFARecoveryCode{
			UserID:    existingUser.ID,
			CodeHash:  string(recHash),
			Used:      false,
			CreatedAt: time.Now(),
		})
	}

	// Seed Sample Local Image files
	_ = os.MkdirAll(filepath.Join(localCfg.BaseDir, "products"), 0755)
	_ = os.MkdirAll(filepath.Join(localCfg.BaseDir, "avatars"), 0755)

	// Seed Sample Products if empty
	var count int64
	primaryDB.Model(&Product{}).Count(&count)
	if count == 0 {
		products := []Product{
			{
				Title:       "Porsche 911 GT3 RS Touring",
				SKU:         "CAR-911-GT3",
				Price:       224500.00,
				Stock:       3,
				CoverImage:  "https://images.unsplash.com/photo-1614162692292-7ac56d7f7f1e?w=800&q=80",
				Gallery:     `["https://images.unsplash.com/photo-1614162692292-7ac56d7f7f1e?w=800&q=80","https://images.unsplash.com/photo-1503376780353-7e6692767b70?w=800&q=80"]`,
				Description: "High-revving 4.0-liter naturally aspirated flat-six engine delivering 518 horsepower.",
				IsPublished: true,
				CreatedAt:   time.Now().Add(-48 * time.Hour),
			},
			{
				Title:       "McLaren 750S Spider",
				SKU:         "CAR-MCL-750",
				Price:       345000.00,
				Stock:       2,
				CoverImage:  "https://images.unsplash.com/photo-1542282088-72c9c27ed0cd?w=800&q=80",
				Gallery:     `["https://images.unsplash.com/photo-1542282088-72c9c27ed0cd?w=800&q=80"]`,
				Description: "Twin-turbocharged 4.0-liter V8 engine paired with a 7-speed seamless shift gearbox.",
				IsPublished: true,
				CreatedAt:   time.Now().Add(-24 * time.Hour),
			},
			{
				Title:       "Specialty Pour-Over Coffee Station",
				SKU:         "CFE-STA-001",
				Price:       389.00,
				Stock:       45,
				CoverImage:  "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=800&q=80",
				Gallery:     `["https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?w=800&q=80"]`,
				Description: "Handcrafted matte black brass gooseneck kettle with precision digital temperature control.",
				IsPublished: true,
				CreatedAt:   time.Now().Add(-12 * time.Hour),
			},
		}
		for _, p := range products {
			primaryDB.Create(&p)
		}
	}

	// Seed Sample Users if empty
	primaryDB.Model(&User{}).Count(&count)
	if count == 0 {
		users := []User{
			{
				Username:  "alex_dev",
				Email:     "alex@example.com",
				Role:      "lead_engineer",
				Avatar:    "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=400&q=80",
				IsActive:  true,
				Bio:       "Backend engineer focusing on distributed Go systems and high-throughput pipelines.",
				CreatedAt: time.Now().Add(-72 * time.Hour),
			},
			{
				Username:  "elena_design",
				Email:     "elena@example.com",
				Role:      "product_designer",
				Avatar:    "https://images.unsplash.com/photo-1517841905240-472988babdf9?w=400&q=80",
				IsActive:  true,
				Bio:       "Crafting dark-mode-first developer tool interfaces with Svelte and Tailwind.",
				CreatedAt: time.Now().Add(-36 * time.Hour),
			},
		}
		for _, u := range users {
			primaryDB.Create(&u)
		}
	}

	// Seed Sample Analytics if empty
	analyticsDB.Model(&EventMetric{}).Count(&count)
	if count == 0 {
		metrics := []EventMetric{
			{MetricName: "http_requests_total", Category: "traffic", Value: 14820, RecordedAt: time.Now().Add(-1 * time.Hour)},
			{MetricName: "database_query_duration_ms", Category: "performance", Value: 2.45, RecordedAt: time.Now().Add(-30 * time.Minute)},
			{MetricName: "active_sessions_count", Category: "auth", Value: 84, RecordedAt: time.Now().Add(-5 * time.Minute)},
		}
		for _, m := range metrics {
			analyticsDB.Create(&m)
		}

		visitors := []VisitorLog{
			{IPAddress: "192.168.1.101", UserAgent: "Mozilla/5.0 Chrome/120.0", Path: "/api/v1/products", StatusCode: 200, CreatedAt: time.Now().Add(-10 * time.Minute)},
			{IPAddress: "10.0.0.42", UserAgent: "Mozilla/5.0 Safari/17.0", Path: "/api/v1/health", StatusCode: 200, CreatedAt: time.Now().Add(-2 * time.Minute)},
		}
		for _, v := range visitors {
			analyticsDB.Create(&v)
		}
	}
}
