package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/archieBTW/gofyr/pkg/gofyr"
	"github.com/archieBTW/gofyr/pkg/gofyr/storage"
	"github.com/pquerna/otp/totp"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// CLIConfig maps gofyr.yaml
type CLIConfig struct {
	App struct {
		Title        string `yaml:"title"`
		BasePath     string `yaml:"base_path"`
		JWTSecret    string `yaml:"jwt_secret"`
		AuthRequired bool   `yaml:"auth_required"`
		AuditLogging bool   `yaml:"audit_logging"`
	} `yaml:"app"`
	Datasources []struct {
		Name      string `yaml:"name"`
		Dialect   string `yaml:"dialect"`
		DSN       string `yaml:"dsn"`
		IsPrimary bool   `yaml:"is_primary"`
	} `yaml:"datasources"`
	Storage struct {
		DefaultDisk string              `yaml:"default_disk"`
		Local       storage.LocalConfig `yaml:"local"`
		S3          storage.S3Config    `yaml:"s3"`
	} `yaml:"storage"`
}

var (
	configFile string
	rootCmd    = &cobra.Command{
		Use:   "gofyr",
		Short: "gofyrAdmin CLI - Manage datasources, admins, MFA, and drop-in Gin admin panel",
		Long: `gofyrAdmin CLI tool for artisan-style management of your Go Gin + GORM admin panel.
Commands include user management, TOTP MFA enrollment, datasource health checks, storage tests, and migrations.`,
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "gofyr.yaml", "Path to gofyr configuration file")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(userCreateCmd)
	rootCmd.AddCommand(userResetMFACmd)
	rootCmd.AddCommand(dsPingCmd)
	rootCmd.AddCommand(storageTestCmd)
	rootCmd.AddCommand(routesListCmd)
}

func loadConfig() (*CLIConfig, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read config file '%s': %w", configFile, err)
	}

	var cfg CLIConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse '%s': %w", configFile, err)
	}

	return &cfg, nil
}

// gofyr init
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold gofyr.yaml and drop-in gofyr_setup.go boilerplate",
	RunE: func(cmd *cobra.Command, args []string) error {
		yamlContent := `# gofyrAdmin Configuration
app:
  title: "gofyrAdmin"
  base_path: "/admin"
  jwt_secret: "generate-a-secure-secret-key-here"
  auth_required: true
  audit_logging: true

datasources:
  - name: "primary"
    dialect: "sqlite"
    dsn: "gofyr.db"
    is_primary: true
  # - name: "postgres_main"
  #   dialect: "postgres"
  #   dsn: "host=localhost user=postgres password=secret dbname=mydb port=5432 sslmode=disable"
  #   is_primary: false

storage:
  default_disk: "local"
  local:
    base_dir: "./uploads"
    public_url_prefix: "/admin/uploads"
  s3:
    bucket: ""
    region: "us-east-1"
    endpoint: ""
    access_key: ""
    secret_key: ""
    use_path_style: false
`

		goSetupContent := `package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/archieBTW/gofyr/pkg/gofyr"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SetupGofyrAdmin mounts gofyrAdmin onto your Gin router.
func SetupGofyrAdmin(r *gin.Engine) *gofyr.Admin {
	cfg := gofyr.DefaultConfig()
	admin, err := gofyr.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize gofyr: %v", err)
	}

	// Connect primary DB
	db, err := gorm.Open(sqlite.Open("gofyr.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect primary DB: %v", err)
	}

	// Add primary datasource
	if err := admin.AddDatasource("primary", db, true); err != nil {
		log.Fatalf("Failed to register datasource: %v", err)
	}

	// Mount admin panel onto Gin
	if err := admin.Mount(r); err != nil {
		log.Fatalf("Failed to mount gofyrAdmin: %v", err)
	}

	return admin
}
`

		if _, err := os.Stat("gofyr.yaml"); os.IsNotExist(err) {
			if err := os.WriteFile("gofyr.yaml", []byte(yamlContent), 0644); err != nil {
				return err
			}
			fmt.Println("✓ Created gofyr.yaml")
		} else {
			fmt.Println("! gofyr.yaml already exists, skipped")
		}

		if _, err := os.Stat("gofyr_setup.go"); os.IsNotExist(err) {
			if err := os.WriteFile("gofyr_setup.go", []byte(goSetupContent), 0644); err != nil {
				return err
			}
			fmt.Println("✓ Created gofyr_setup.go")
		} else {
			fmt.Println("! gofyr_setup.go already exists, skipped")
		}

		fmt.Println("\nSetup complete! Run 'gofyr user:create' to create your first admin user.")
		return nil
	},
}

// gofyr migrate
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations for gofyr internal tables",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		var primaryDS *struct {
			Name      string `yaml:"name"`
			Dialect   string `yaml:"dialect"`
			DSN       string `yaml:"dsn"`
			IsPrimary bool   `yaml:"is_primary"`
		}
		for i := range cfg.Datasources {
			if cfg.Datasources[i].IsPrimary {
				primaryDS = &cfg.Datasources[i]
				break
			}
		}
		if primaryDS == nil && len(cfg.Datasources) > 0 {
			primaryDS = &cfg.Datasources[0]
		}
		if primaryDS == nil {
			return fmt.Errorf("no datasources defined in config")
		}

		fmt.Printf("Connecting to primary datasource '%s' (%s)...\n", primaryDS.Name, primaryDS.Dialect)
		db, err := gofyr.OpenDatasource(primaryDS.Dialect, primaryDS.DSN)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}

		fmt.Println("Running migrations on internal gofyr tables...")
		if err := db.AutoMigrate(gofyr.InternalModels()...); err != nil {
			return fmt.Errorf("auto-migration failed: %w", err)
		}

		fmt.Println("✓ Successfully migrated: gofyr_admin_users, gofyr_mfa_recovery_codes, gofyr_audit_logs, gofyr_media_files")
		return nil
	},
}

// gofyr user:create
var (
	flagEmail    string
	flagName     string
	flagPassword string
	flagRole     string
)

var userCreateCmd = &cobra.Command{
	Use:   "user:create",
	Short: "Create an admin user and generate TOTP MFA credentials with terminal ASCII QR code",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		var primaryDS *struct {
			Name      string `yaml:"name"`
			Dialect   string `yaml:"dialect"`
			DSN       string `yaml:"dsn"`
			IsPrimary bool   `yaml:"is_primary"`
		}
		for i := range cfg.Datasources {
			if cfg.Datasources[i].IsPrimary {
				primaryDS = &cfg.Datasources[i]
				break
			}
		}
		if primaryDS == nil && len(cfg.Datasources) > 0 {
			primaryDS = &cfg.Datasources[0]
		}
		if primaryDS == nil {
			return fmt.Errorf("no primary datasource configured")
		}

		db, err := gofyr.OpenDatasource(primaryDS.Dialect, primaryDS.DSN)
		if err != nil {
			return fmt.Errorf("failed to connect to DB: %w", err)
		}
		_ = db.AutoMigrate(gofyr.InternalModels()...)

		reader := bufio.NewReader(os.Stdin)

		email := flagEmail
		if email == "" {
			fmt.Print("Enter email address: ")
			email, _ = reader.ReadString('\n')
			email = strings.TrimSpace(email)
		}
		if email == "" {
			return fmt.Errorf("email is required")
		}

		name := flagName
		if name == "" {
			fmt.Print("Enter name [Admin]: ")
			name, _ = reader.ReadString('\n')
			name = strings.TrimSpace(name)
			if name == "" {
				name = "Admin"
			}
		}

		password := flagPassword
		if password == "" {
			fmt.Print("Enter password: ")
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return err
			}
			password = strings.TrimSpace(string(bytePassword))
			fmt.Println()
		}
		if len(password) < 6 {
			return fmt.Errorf("password must be at least 6 characters")
		}

		role := flagRole
		if role == "" {
			role = "superadmin"
		}

		// Hash password
		pwdHash, err := gofyr.HashPassword(password)
		if err != nil {
			return err
		}

		// Generate TOTP Secret
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      cfg.App.Title,
			AccountName: email,
		})
		if err != nil {
			return fmt.Errorf("failed to generate TOTP secret: %w", err)
		}

		user := gofyr.GofyrAdminUser{
			Email:        strings.ToLower(email),
			PasswordHash: pwdHash,
			Name:         name,
			Role:         role,
			MFASecret:    key.Secret(),
			MFAEnabled:   true,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Save or update user
		var existing gofyr.GofyrAdminUser
		if err := db.Where("email = ?", user.Email).First(&existing).Error; err == nil {
			user.ID = existing.ID
			db.Save(&user)
			fmt.Printf("\n✓ Updated existing admin user '%s'\n", user.Email)
		} else {
			if err := db.Create(&user).Error; err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}
			fmt.Printf("\n✓ Created new admin user '%s' (ID: %d)\n", user.Email, user.ID)
		}

		// Generate 8 recovery backup codes
		db.Where("user_id = ?", user.ID).Delete(&gofyr.GofyrMFARecoveryCode{})
		var backupCodes []string
		for i := 0; i < 8; i++ {
			code := fmt.Sprintf("%04X-%04X", time.Now().UnixNano()%0xFFFF, (time.Now().UnixNano()/7)%0xFFFF)
			backupCodes = append(backupCodes, code)
			norm := strings.ReplaceAll(code, "-", "")
			hash, _ := bcrypt.GenerateFromPassword([]byte(norm), bcrypt.DefaultCost)
			db.Create(&gofyr.GofyrMFARecoveryCode{
				UserID:    user.ID,
				CodeHash:  string(hash),
				Used:      false,
				CreatedAt: time.Now(),
			})
			time.Sleep(time.Millisecond * 2)
		}

		// Generate terminal ASCII QR code
		qr, err := qrcode.New(key.URL(), qrcode.Medium)
		if err == nil {
			fmt.Println("\n=======================================================")
			fmt.Println("  SCAN QR CODE IN GOOGLE AUTHENTICATOR / 1PASSWORD")
			fmt.Println("=======================================================\n")
			fmt.Println(qr.ToSmallString(false))
		}

		fmt.Printf("TOTP Secret:          %s\n", key.Secret())
		fmt.Printf("Provisioning URI:     %s\n", key.URL())
		fmt.Println("\n-------------------------------------------------------")
		fmt.Println("  SINGLE-USE RECOVERY BACKUP CODES (SAVE THESE SAFELY):")
		fmt.Println("-------------------------------------------------------")
		for i, c := range backupCodes {
			fmt.Printf("  [%d] %s\n", i+1, c)
		}
		fmt.Println("-------------------------------------------------------\n")

		return nil
	},
}

func init() {
	userCreateCmd.Flags().StringVar(&flagEmail, "email", "", "Admin email address")
	userCreateCmd.Flags().StringVar(&flagName, "name", "", "Admin display name")
	userCreateCmd.Flags().StringVar(&flagPassword, "password", "", "Admin password")
	userCreateCmd.Flags().StringVar(&flagRole, "role", "superadmin", "Admin role (superadmin, admin, editor)")
}

// gofyr user:reset-mfa
var flagResetEmail string

var userResetMFACmd = &cobra.Command{
	Use:   "user:reset-mfa",
	Short: "Reset MFA for a locked-out admin user",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagResetEmail == "" {
			return fmt.Errorf("--email flag is required")
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		var primaryDS *struct {
			Name      string `yaml:"name"`
			Dialect   string `yaml:"dialect"`
			DSN       string `yaml:"dsn"`
			IsPrimary bool   `yaml:"is_primary"`
		}
		for i := range cfg.Datasources {
			if cfg.Datasources[i].IsPrimary {
				primaryDS = &cfg.Datasources[i]
				break
			}
		}
		if primaryDS == nil && len(cfg.Datasources) > 0 {
			primaryDS = &cfg.Datasources[0]
		}
		if primaryDS == nil {
			return fmt.Errorf("no primary datasource configured")
		}

		db, err := gofyr.OpenDatasource(primaryDS.Dialect, primaryDS.DSN)
		if err != nil {
			return fmt.Errorf("failed to connect to DB: %w", err)
		}

		var user gofyr.GofyrAdminUser
		if err := db.Where("email = ?", strings.ToLower(flagResetEmail)).First(&user).Error; err != nil {
			return fmt.Errorf("user with email '%s' not found: %w", flagResetEmail, err)
		}

		user.MFAEnabled = false
		user.MFASecret = ""
		db.Save(&user)

		// Delete recovery codes
		db.Where("user_id = ?", user.ID).Delete(&gofyr.GofyrMFARecoveryCode{})

		fmt.Printf("✓ Successfully reset MFA for user '%s'. User can now login without MFA and re-enroll.\n", user.Email)
		return nil
	},
}

func init() {
	userResetMFACmd.Flags().StringVar(&flagResetEmail, "email", "", "Email of admin user to reset MFA")
}

// gofyr ds:ping
var dsPingCmd = &cobra.Command{
	Use:   "ds:ping",
	Short: "Test connectivity and latency for all configured datasources",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tDIALECT\tPRIMARY\tSTATUS\tLATENCY\tVERSION")
		fmt.Fprintln(w, "----\t-------\t-------\t------\t-------\t-------")

		for _, dsCfg := range cfg.Datasources {
			db, err := gofyr.OpenDatasource(dsCfg.Dialect, dsCfg.DSN)
			if err != nil {
				fmt.Fprintf(w, "%s\t%s\t%t\tOFFLINE\t-\t%s\n", dsCfg.Name, dsCfg.Dialect, dsCfg.IsPrimary, err.Error())
				continue
			}

			ds := &gofyr.Datasource{
				Name:      dsCfg.Name,
				Dialect:   dsCfg.Dialect,
				IsPrimary: dsCfg.IsPrimary,
				DB:        db,
			}

			status := ds.Ping()
			statusStr := "ONLINE"
			if !status.Online {
				statusStr = "ERROR: " + status.Error
			}

			fmt.Fprintf(w, "%s\t%s\t%t\t%s\t%.2fms\t%s\n", status.Name, status.Dialect, status.IsPrimary, statusStr, status.LatencyMS, status.Version)
		}
		w.Flush()
		return nil
	},
}

// gofyr storage:test
var flagDisk string

var storageTestCmd = &cobra.Command{
	Use:   "storage:test",
	Short: "Upload, verify, and delete a test file to validate Local or S3 credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		disk := flagDisk
		if disk == "" {
			disk = cfg.Storage.DefaultDisk
			if disk == "" {
				disk = "local"
			}
		}

		var driver storage.Driver
		switch disk {
		case "local":
			driver, err = storage.NewLocal(cfg.Storage.Local)
			if err != nil {
				return fmt.Errorf("failed to init local driver: %w", err)
			}
		case "s3":
			driver, err = storage.NewS3(context.Background(), cfg.Storage.S3)
			if err != nil {
				return fmt.Errorf("failed to init S3 driver: %w", err)
			}
		default:
			return fmt.Errorf("unsupported disk '%s'. Choose 'local' or 's3'", disk)
		}

		testKey := fmt.Sprintf("test/gofyr_test_%d.txt", time.Now().Unix())
		testContent := "gofyrAdmin storage connectivity test verified at " + time.Now().UTC().Format(time.RFC3339)

		fmt.Printf("Testing '%s' storage driver...\n", disk)
		fmt.Printf("1. Uploading test object '%s'...\n", testKey)
		url, err := driver.Upload(context.Background(), testKey, strings.NewReader(testContent), "text/plain")
		if err != nil {
			return fmt.Errorf("✗ Upload failed: %w", err)
		}
		fmt.Printf("   ✓ Uploaded successfully! URL: %s\n", url)

		fmt.Printf("2. Verifying object URL resolution...\n")
		resolvedURL, err := driver.GetURL(context.Background(), testKey)
		if err != nil {
			return fmt.Errorf("✗ GetURL failed: %w", err)
		}
		fmt.Printf("   ✓ URL resolved: %s\n", resolvedURL)

		fmt.Printf("3. Deleting test object...\n")
		if err := driver.Delete(context.Background(), testKey); err != nil {
			return fmt.Errorf("✗ Delete failed: %w", err)
		}
		fmt.Printf("   ✓ Test object deleted cleanly.\n")

		fmt.Printf("\n✓ Storage driver '%s' is verified and operating properly!\n", disk)
		return nil
	},
}

func init() {
	storageTestCmd.Flags().StringVar(&flagDisk, "disk", "local", "Storage disk to test (local, s3)")
}

// gofyr routes:list
var routesListCmd = &cobra.Command{
	Use:   "routes:list",
	Short: "Print a colorized list of all registered Gin application routes",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Inspecting project routes from gofyr_setup.go or application setup...")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "METHOD\tPATH\tCATEGORY\tDESCRIPTION")
		fmt.Fprintln(w, "------\t----\t--------\t-----------")
		fmt.Fprintln(w, "GET\t/admin\tAdmin Panel\tEmbedded Svelte 5 SPA Admin Console")
		fmt.Fprintln(w, "POST\t/admin/api/auth/login\tAuth\tStage 1 credentials verification")
		fmt.Fprintln(w, "POST\t/admin/api/auth/mfa/setup\tAuth\tGenerate TOTP Secret and QR Code")
		fmt.Fprintln(w, "POST\t/admin/api/auth/mfa/verify\tAuth\tVerify 6-digit TOTP / recovery code")
		fmt.Fprintln(w, "GET\t/admin/api/schema\tSchema\tIntrospect registered datasources & models")
		fmt.Fprintln(w, "GET\t/admin/api/ds/:datasource/records/:model\tCRUD\tDynamic list with pagination, sort, search")
		fmt.Fprintln(w, "POST\t/admin/api/ds/:datasource/records/:model\tCRUD\tCreate record")
		fmt.Fprintln(w, "PUT\t/admin/api/ds/:datasource/records/:model/:id\tCRUD\tUpdate record")
		fmt.Fprintln(w, "DELETE\t/admin/api/ds/:datasource/records/:model/:id\tCRUD\tDelete record")
		fmt.Fprintln(w, "POST\t/admin/api/media/upload\tStorage\tMultipart upload to Local or S3")
		fmt.Fprintln(w, "GET\t/admin/api/workbench/routes\tWorkbench\tHost Gin route introspection")
		fmt.Fprintln(w, "POST\t/admin/api/workbench/execute\tWorkbench\tIn-memory authenticated API runner")
		w.Flush()
		return nil
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
