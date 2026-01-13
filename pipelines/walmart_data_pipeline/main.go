package main

import (
	"archive/zip"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// KaggleCredentials holds the authentication details.
type KaggleCredentials struct {
	Username string `json:"username"`
	Key      string `json:"key"`
}

// WalmartSale represents a single row from the dataset.
type WalmartSale struct {
	Store        int
	Date         time.Time
	WeeklySales  float64
	HolidayFlag  bool
	Temperature  float64
	FuelPrice    float64
	CPI          float64
	Unemployment float64
}

type Config struct {
	KaggleCredentialsPath string
	DatasetURL            string
	RawDataDir            string
	DBHost                string
	DBPort                string
	DBUser                string
	DBPassword            string
	DBName                string
}

func main() {
	repoRoot := findRepoRoot()

	// Load repo-root .env if present (non-fatal).
	// Existing environment variables are NOT overridden by godotenv.Load().
	_ = godotenv.Load(filepath.Join(repoRoot, ".env"))

	defaultCreds := filepath.Join(repoRoot, ".secrets", "kaggle.json")
	defaultRaw := filepath.Join(repoRoot, "data", "raw", "walmart")

	config := Config{
		KaggleCredentialsPath: getEnv("KAGGLE_CREDENTIALS_PATH", defaultCreds),
		DatasetURL:            getEnv("DATASET_URL", "https://www.kaggle.com/api/v1/datasets/download/yasserh/walmart-dataset"),
		RawDataDir:            getEnv("RAW_DATA_DIR", defaultRaw),
		DBHost:                getEnv("DB_HOST", "localhost"),
		DBPort:                getEnv("DB_PORT", "5433"),
		DBUser:                getEnv("DB_USER", "walmart_user"),
		DBPassword:            getEnv("DB_PASSWORD", "walmart_pass"),
		DBName:                getEnv("DB_NAME", "walmart_db"),
	}

	log.Println("=== Walmart Data Pipeline ===")
	log.Printf("Repo root: %s", repoRoot)
	log.Printf("Raw data dir: %s", config.RawDataDir)

	// Step 1: Load Kaggle credentials.
	log.Println("Step 1: Loading Kaggle credentials...")
	creds, err := loadKaggleCredentials(config.KaggleCredentialsPath)
	if err != nil {
		log.Fatalf("Failed to load Kaggle credentials: %v", err)
	}
	log.Printf("Credentials loaded for user: %s", creds.Username)

	// Step 2: Download dataset.
	log.Println("Step 2: Downloading dataset from Kaggle...")
	zipPath, err := downloadDataset(config.DatasetURL, config.RawDataDir, creds)
	if err != nil {
		log.Fatalf("Failed to download dataset: %v", err)
	}
	log.Printf("Dataset downloaded to: %s", zipPath)

	// Step 3: Extract the ZIP file.
	log.Println("Step 3: Extracting dataset...")
	csvPath, err := extractZip(zipPath, config.RawDataDir)
	if err != nil {
		log.Fatalf("Failed to extract dataset: %v", err)
	}
	log.Printf("Dataset extracted; CSV found at: %s", csvPath)

	// Step 4: Parse the CSV data.
	log.Println("Step 4: Parsing CSV data...")
	sales, err := parseCSV(csvPath)
	if err != nil {
		log.Fatalf("Failed to parse CSV: %v", err)
	}
	log.Printf("Parsed %d records", len(sales))

	// Step 5: Connect to PostgreSQL.
	log.Println("Step 5: Connecting to PostgreSQL...")
	db, err := connectDB(config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to PostgreSQL")

	// Step 6: Import data.
	log.Println("Step 6: Importing data to PostgreSQL...")
	count, err := importData(db, sales)
	if err != nil {
		log.Fatalf("Failed to import data: %v", err)
	}
	log.Printf("Successfully imported %d records", count)

	log.Println("=== Pipeline completed successfully! ===")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// findRepoRoot attempts to locate the repository root directory.
// It searches upward from the current working directory for a marker file (pyproject.toml or .git).
func findRepoRoot() string {
	cwd, err := os.Getwd()
	if err == nil {
		if root, ok := walkUpForMarker(cwd); ok {
			return root
		}
	}

	// Fallback: derive from executable location.
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	exeDir := filepath.Dir(exe)
	if root, ok := walkUpForMarker(exeDir); ok {
		return root
	}

	return "."
}

func walkUpForMarker(start string) (string, bool) {
	dir := filepath.Clean(start)
	for i := 0; i < 10; i++ {
		if fileExists(filepath.Join(dir, "pyproject.toml")) || dirExists(filepath.Join(dir, ".git")) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func loadKaggleCredentials(path string) (*KaggleCredentials, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open credentials file: %w", err)
	}
	defer file.Close()

	var creds KaggleCredentials
	if err := json.NewDecoder(file).Decode(&creds); err != nil {
		return nil, fmt.Errorf("failed to decode credentials JSON: %w", err)
	}

	if creds.Username == "" || creds.Key == "" {
		return nil, fmt.Errorf("credentials file is missing username or key")
	}

	return &creds, nil
}

func downloadDataset(url, rawDataDir string, creds *KaggleCredentials) (string, error) {
	if err := os.MkdirAll(rawDataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create raw data directory: %w", err)
	}

	zipPath := filepath.Join(rawDataDir, "walmart-dataset.zip")

	// Kaggle API URL download via Basic Auth (username:key).
	cmd := exec.Command("curl",
		"-L",
		"-o", zipPath,
		"-u", fmt.Sprintf("%s:%s", creds.Username, creds.Key),
		"--fail",
		"--silent",
		"--show-error",
		url,
	)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to download dataset: %w", err)
	}

	// Basic sanity check: ensure file exists and isn't trivially small.
	if info, err := os.Stat(zipPath); err != nil {
		return "", fmt.Errorf("downloaded zip not found: %w", err)
	} else if info.Size() < 10_000 {
		return "", fmt.Errorf("downloaded zip looks too small (%d bytes); check Kaggle credentials/URL", info.Size())
	}

	return zipPath, nil
}

func extractZip(zipPath, extractPath string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	var csvPath string

	for _, f := range r.File {
		destPath := filepath.Join(extractPath, f.Name)

		// Prevent zip-slip / path traversal.
		if !strings.HasPrefix(destPath, filepath.Clean(extractPath)+string(os.PathSeparator)) {
			return "", fmt.Errorf("invalid file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, f.Mode()); err != nil {
				return "", fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}

		src, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("failed to open file in zip: %w", err)
		}
		defer src.Close()

		dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", fmt.Errorf("failed to create extracted file: %w", err)
		}

		if _, err := io.Copy(dst, src); err != nil {
			_ = dst.Close()
			return "", fmt.Errorf("failed to extract file: %w", err)
		}
		_ = dst.Close()

		if strings.HasSuffix(strings.ToLower(destPath), ".csv") {
			csvPath = destPath
		}
	}

	if csvPath == "" {
		return "", fmt.Errorf("no CSV file found in zip")
	}

	return csvPath, nil
}

func parseCSV(csvPath string) ([]WalmartSale, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header (and discard).
	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	sales := make([]WalmartSale, 0, 1024)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Warning: skipping unreadable record: %v", err)
			continue
		}

		sale, err := parseRecord(record)
		if err != nil {
			log.Printf("Warning: skipping invalid record: %v", err)
			continue
		}

		sales = append(sales, sale)
	}

	return sales, nil
}

func parseRecord(record []string) (WalmartSale, error) {
	if len(record) < 8 {
		return WalmartSale{}, fmt.Errorf("record has insufficient columns: %d", len(record))
	}

	store, err := strconv.Atoi(strings.TrimSpace(record[0]))
	if err != nil {
		return WalmartSale{}, fmt.Errorf("invalid store: %w", err)
	}

	rawDate := strings.TrimSpace(record[1])
	dt, err := parseDate(rawDate)
	if err != nil {
		return WalmartSale{}, fmt.Errorf("invalid date '%s': %w", rawDate, err)
	}

	weeklySales, err := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)
	if err != nil {
		return WalmartSale{}, fmt.Errorf("invalid weekly_sales: %w", err)
	}

	holidayFlag := strings.TrimSpace(record[3]) == "1"

	temperature, err := strconv.ParseFloat(strings.TrimSpace(record[4]), 64)
	if err != nil {
		return WalmartSale{}, fmt.Errorf("invalid temperature: %w", err)
	}

	fuelPrice, err := strconv.ParseFloat(strings.TrimSpace(record[5]), 64)
	if err != nil {
		return WalmartSale{}, fmt.Errorf("invalid fuel_price: %w", err)
	}

	cpi, err := strconv.ParseFloat(strings.TrimSpace(record[6]), 64)
	if err != nil {
		return WalmartSale{}, fmt.Errorf("invalid cpi: %w", err)
	}

	unemployment, err := strconv.ParseFloat(strings.TrimSpace(record[7]), 64)
	if err != nil {
		return WalmartSale{}, fmt.Errorf("invalid unemployment: %w", err)
	}

	return WalmartSale{
		Store:        store,
		Date:         dt,
		WeeklySales:  weeklySales,
		HolidayFlag:  holidayFlag,
		Temperature:  temperature,
		FuelPrice:    fuelPrice,
		CPI:          cpi,
		Unemployment: unemployment,
	}, nil
}

func parseDate(s string) (time.Time, error) {
	layouts := []string{
		"02-01-2006", // DD-MM-YYYY
		"2006-01-02", // YYYY-MM-DD
		"1/2/2006",   // M/D/YYYY
		"01/02/2006", // MM/DD/YYYY
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported date format: %s", s)
}

func connectDB(config Config) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func importData(db *sql.DB, sales []WalmartSale) (int, error) {
	stmt, err := db.Prepare(`
		INSERT INTO walmart_sales (store, date, weekly_sales, holiday_flag, temperature, fuel_price, cpi, unemployment)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	count := 0
	for _, sale := range sales {
		_, err := tx.Stmt(stmt).Exec(
			sale.Store,
			sale.Date,
			sale.WeeklySales,
			sale.HolidayFlag,
			sale.Temperature,
			sale.FuelPrice,
			sale.CPI,
			sale.Unemployment,
		)
		if err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("failed to insert record: %w", err)
		}

		count++
		if count%1000 == 0 {
			log.Printf("Imported %d records...", count)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}
