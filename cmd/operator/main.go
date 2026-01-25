// Package main provides the operator CLI for deployment and operations tasks.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/easeaico/project-her/internal/config"
	"google.golang.org/adk/session/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "migrate":
		migrateCmd(os.Args[2:])
	case "schema":
		schemaCmd(os.Args[2:])
	case "validate":
		validateCmd()
	case "version":
		fmt.Printf("project-her operator v%s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`project-her operator - Deployment and operations CLI

Usage:
  operator <command> [flags]

Commands:
  migrate     Run database migrations (ADK session tables and app tables)
  schema      Execute SQL migration files from migrations/ directory
  validate    Validate environment configuration
  version     Show version information
  help        Show this help message

Examples:
  operator migrate              # Run all migrations
  operator migrate --adk-only   # Only migrate ADK session tables
  operator migrate --app-only   # Only migrate app tables (GORM AutoMigrate)
  operator schema               # Execute all SQL files in migrations/
  operator schema --file 001_init.sql  # Execute a specific migration file
  operator validate             # Check if all required env vars are set`)
}

// migrateCmd handles the migrate command.
func migrateCmd(args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	adkOnly := fs.Bool("adk-only", false, "Only migrate ADK session tables")
	appOnly := fs.Bool("app-only", false, "Only migrate application tables")
	dryRun := fs.Bool("dry-run", false, "Show what would be migrated without executing")
	_ = fs.Parse(args)

	if *adkOnly && *appOnly {
		log.Fatal("cannot use both --adk-only and --app-only")
	}

	cfg := loadConfigForOperator()

	if *dryRun {
		fmt.Println("Dry run mode - no changes will be made")
		if !*appOnly {
			fmt.Println("  - Would migrate ADK session tables")
		}
		if !*adkOnly {
			fmt.Println("  - Would migrate application tables (characters, memories, chat_histories)")
		}
		return
	}

	// Migrate ADK session tables
	if !*appOnly {
		fmt.Println("Migrating ADK session tables...")
		if err := migrateADKSession(cfg.DatabaseURL); err != nil {
			log.Fatalf("failed to migrate ADK session: %v", err)
		}
		fmt.Println("  ✓ ADK session tables migrated")
	}

	// Migrate application tables
	if !*adkOnly {
		fmt.Println("Migrating application tables...")
		if err := migrateAppTables(cfg.DatabaseURL); err != nil {
			log.Fatalf("failed to migrate app tables: %v", err)
		}
		fmt.Println("  ✓ Application tables migrated")
	}

	fmt.Println("\nMigration completed successfully!")
}

// schemaCmd handles the schema command for executing SQL files.
func schemaCmd(args []string) {
	fs := flag.NewFlagSet("schema", flag.ExitOnError)
	file := fs.String("file", "", "Specific migration file to execute")
	migrationsDir := fs.String("dir", "migrations", "Directory containing migration files")
	dryRun := fs.Bool("dry-run", false, "Show what would be executed without running")
	_ = fs.Parse(args)

	cfg := loadConfigForOperator()

	// Find migration files
	files, err := findMigrationFiles(*migrationsDir, *file)
	if err != nil {
		log.Fatalf("failed to find migration files: %v", err)
	}

	if len(files) == 0 {
		fmt.Println("No migration files found")
		return
	}

	fmt.Printf("Found %d migration file(s):\n", len(files))
	for _, f := range files {
		fmt.Printf("  - %s\n", filepath.Base(f))
	}

	if *dryRun {
		fmt.Println("\nDry run mode - no SQL will be executed")
		return
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql DB handle: %v", err)
	}
	defer func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			log.Printf("failed to close database connection: %v", closeErr)
		}
	}()

	// Execute each migration
	fmt.Println("\nExecuting migrations...")
	for _, f := range files {
		fmt.Printf("  Running %s... ", filepath.Base(f))
		if err := executeSQLFile(db, f); err != nil {
			fmt.Println("✗")
			log.Fatalf("failed to execute %s: %v", f, err)
		}
		fmt.Println("✓")
	}

	fmt.Println("\nSchema migration completed successfully!")
}

// validateCmd validates the configuration.
func validateCmd() {
	fmt.Println("Validating configuration...")

	required := []struct {
		name     string
		envVar   string
		required bool
	}{
		{"Database URL", "DATABASE_URL", true},
		{"Google API Key", "GOOGLE_API_KEY", true},
		{"xAI API Key", "XAI_API_KEY", true},
		{"Work Directory", "WORK_DIR", false},
		{"Chat Model", "CHAT_MODEL", false},
		{"Memory Model", "MEMORY_MODEL", false},
		{"Image Model", "IMAGE_MODEL", false},
		{"Embedding Model", "EMBEDDING_MODEL", false},
		{"Image Storage Dir", "IMAGE_STORAGE_DIR", false},
		{"Image Base URL", "IMAGE_BASE_URL", false},
	}

	hasErrors := false
	for _, r := range required {
		value := os.Getenv(r.envVar)
		if value == "" {
			if r.required {
				fmt.Printf("  ✗ %s (%s): NOT SET (required)\n", r.name, r.envVar)
				hasErrors = true
			} else {
				fmt.Printf("  - %s (%s): not set (optional, will use default)\n", r.name, r.envVar)
			}
		} else {
			// Mask sensitive values
			displayValue := value
			if strings.Contains(strings.ToLower(r.envVar), "key") ||
				strings.Contains(strings.ToLower(r.envVar), "password") ||
				strings.Contains(strings.ToLower(r.envVar), "secret") {
				displayValue = maskValue(value)
			}
			if strings.Contains(strings.ToLower(r.envVar), "url") && strings.Contains(value, "@") {
				displayValue = maskDatabaseURL(value)
			}
			fmt.Printf("  ✓ %s (%s): %s\n", r.name, r.envVar, displayValue)
		}
	}

	if hasErrors {
		fmt.Println("\nConfiguration validation failed!")
		os.Exit(1)
	}

	// Test database connection
	fmt.Println("\nTesting database connection...")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
		if err != nil {
			fmt.Printf("  ✗ Failed to connect: %v\n", err)
			os.Exit(1)
		}
		sqlDB, err := db.DB()
		if err != nil {
			fmt.Printf("  ✗ Failed to get connection: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			if closeErr := sqlDB.Close(); closeErr != nil {
				fmt.Printf("  ! Failed to close database connection: %v\n", closeErr)
			}
		}()

		if err := sqlDB.PingContext(ctx); err != nil {
			fmt.Printf("  ✗ Failed to ping: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("  ✓ Database connection successful")

		// Check for pgvector extension
		var extExists bool
		db.Raw("SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'vector')").Scan(&extExists)
		if extExists {
			fmt.Println("  ✓ pgvector extension installed")
		} else {
			fmt.Println("  ! pgvector extension not installed (required for memory features)")
		}
	}

	fmt.Println("\nConfiguration validation completed!")
}

// loadConfigForOperator loads config with relaxed validation for operator commands.
func loadConfigForOperator() config.Config {
	// For operator, we only need DATABASE_URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	return config.Config{
		DatabaseURL: dbURL,
	}
}

func migrateADKSession(databaseURL string) error {
	sessionService, err := database.NewSessionService(postgres.Open(databaseURL))
	if err != nil {
		return fmt.Errorf("failed to create session service: %w", err)
	}

	if err := database.AutoMigrate(sessionService); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	return nil
}

func migrateAppTables(databaseURL string) error {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql DB handle: %w", err)
	}
	defer func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			log.Printf("failed to close database connection: %v", closeErr)
		}
	}()

	// AutoMigrate will create tables, missing foreign keys, constraints, columns and indexes.
	// Note: For pgvector columns and complex indexes, use SQL migrations instead.
	type Character struct {
		ID              uint   `gorm:"primaryKey"`
		Name            string `gorm:"size:255;not null"`
		Description     string `gorm:"type:text"`
		Appearance      string `gorm:"type:text"`
		Personality     string `gorm:"type:text"`
		Scenario        string `gorm:"type:text"`
		FirstMessage    string `gorm:"type:text"`
		ExampleDialogue string `gorm:"type:text"`
		SystemPrompt    string `gorm:"type:text"`
		AvatarPath      string `gorm:"size:255"`
		Affection       int    `gorm:"default:50"`
		CurrentMood     string `gorm:"size:50;default:'Neutral'"`
		CreatedAt       time.Time
		UpdatedAt       time.Time
	}

	type ChatHistory struct {
		ID          uint   `gorm:"primaryKey"`
		UserID      string `gorm:"size:64"`
		AppName     string `gorm:"size:255"`
		CharacterID uint
		Content     string `gorm:"type:text;not null"`
		TurnCount   int    `gorm:"default:0"`
		Summarized  bool   `gorm:"default:false"`
		CreatedAt   time.Time
	}

	// Note: Memory table requires pgvector, so we skip AutoMigrate for it
	// and rely on SQL migrations for the full schema.

	if err := db.AutoMigrate(&Character{}, &ChatHistory{}); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	return nil
}

func findMigrationFiles(dir, specificFile string) ([]string, error) {
	if specificFile != "" {
		fullPath := filepath.Join(dir, specificFile)
		if _, err := os.Stat(fullPath); err != nil {
			return nil, fmt.Errorf("migration file not found: %s", fullPath)
		}
		return []string{fullPath}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	// Sort by filename to ensure consistent ordering
	sort.Strings(files)
	return files, nil
}

func executeSQLFile(db *gorm.DB, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Execute the SQL
	if err := db.Exec(string(content)).Error; err != nil {
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	return nil
}

func maskValue(value string) string {
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

func maskDatabaseURL(url string) string {
	// Mask password in postgres://user:password@host:port/db format
	atIndex := strings.Index(url, "@")
	if atIndex == -1 {
		return url
	}
	prefix := url[:strings.Index(url, "://")+3]
	remainder := url[len(prefix):]

	colonIndex := strings.Index(remainder, ":")
	atInRemainder := strings.Index(remainder, "@")

	if colonIndex != -1 && colonIndex < atInRemainder {
		user := remainder[:colonIndex]
		afterAt := remainder[atInRemainder:]
		return prefix + user + ":****" + afterAt
	}
	return url
}
