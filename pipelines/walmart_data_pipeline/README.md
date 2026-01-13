# Walmart Data Pipeline

A Go application that extracts Walmart sales data from Kaggle and imports it into PostgreSQL.

## Prerequisites

- **Go** 1.21 or later
- **Docker** and Docker Compose
- **curl** (usually pre-installed on most systems)
- **Kaggle account** with API credentials

## Project Structure

```
walmart-data-pipeline/
├── credentials/
│   └── kaggle.json          # Your Kaggle API credentials
├── data/                    # Downloaded and extracted data (created at runtime)
├── docker-compose.yml       # PostgreSQL container configuration
├── init.sql                 # Database schema initialization
├── go.mod                   # Go module definition
├── main.go                  # Main application code
├── run_pipeline.sh          # Automated setup and run script
└── README.md                # This file
```

## Setup Instructions

### 1. Configure Kaggle Credentials

1. Go to [Kaggle Settings](https://www.kaggle.com/settings/account)
2. Scroll to the **API** section
3. Click **Create New Token** (this downloads a `kaggle.json` file)
4. Copy the contents to `credentials/kaggle.json`

The file should look like:
```json
{
    "username": "your_kaggle_username",
    "key": "your_kaggle_api_key"
}
```

### 2. Run the Pipeline

Make the script executable and run it:

```bash
chmod +x run_pipeline.sh
./run_pipeline.sh
```

This script will:
1. Check that Docker is running
2. Pull the latest PostgreSQL image
3. Start the PostgreSQL container
4. Build and run the Go application
5. Download, extract, and import the Walmart dataset

### 3. Manual Execution (Alternative)

If you prefer to run steps manually:

```bash
# Start PostgreSQL
docker-compose up -d

# Wait for PostgreSQL to be ready
docker exec walmart_postgres pg_isready -U walmart_user -d walmart_db

# Build and run the Go application
go build -o walmart-pipeline main.go
./walmart-pipeline
```

## Database Connection

- **Host:** localhost
- **Port:** 5432
- **Database:** walmart_db
- **User:** walmart_user
- **Password:** walmart_pass

### Connect via psql

```bash
docker exec -it walmart_postgres psql -U walmart_user -d walmart_db
```

### Example Queries

```sql
-- Count total records
SELECT COUNT(*) FROM walmart_sales;

-- View sample data
SELECT * FROM walmart_sales LIMIT 10;

-- Average weekly sales by store
SELECT store, ROUND(AVG(weekly_sales)::numeric, 2) as avg_sales 
FROM walmart_sales 
GROUP BY store 
ORDER BY store;

-- Sales on holidays vs non-holidays
SELECT 
    holiday_flag,
    COUNT(*) as count,
    ROUND(AVG(weekly_sales)::numeric, 2) as avg_sales
FROM walmart_sales 
GROUP BY holiday_flag;

-- Monthly sales trend
SELECT 
    SUBSTRING(date, 4, 2) as month,
    ROUND(SUM(weekly_sales)::numeric, 2) as total_sales
FROM walmart_sales 
GROUP BY SUBSTRING(date, 4, 2)
ORDER BY month;
```

## Schema

The `walmart_sales` table has the following structure:

| Column       | Type          | Description                  |
| ------------ | ------------- | ---------------------------- |
| id           | SERIAL        | Primary key (auto-generated) |
| store        | INTEGER       | Store number                 |
| date         | VARCHAR(20)   | Date of the sales record     |
| weekly_sales | DECIMAL(12,2) | Weekly sales amount          |
| holiday_flag | BOOLEAN       | Whether it's a holiday week  |
| temperature  | DECIMAL(6,2)  | Temperature in the region    |
| fuel_price   | DECIMAL(6,3)  | Fuel price in the region     |
| cpi          | DECIMAL(12,6) | Consumer Price Index         |
| unemployment | DECIMAL(6,3)  | Unemployment rate            |
| created_at   | TIMESTAMP     | Record creation timestamp    |

## Cleanup

To stop and remove the PostgreSQL container:

```bash
docker-compose down
```

To also remove the data volume:

```bash
docker-compose down -v
```

## Troubleshooting

### "Connection refused" error
Make sure PostgreSQL is running:
```bash
docker-compose ps
docker logs walmart_postgres
```

### Kaggle authentication failed
- Verify your credentials in `credentials/kaggle.json`
- Make sure you've accepted the dataset's terms on Kaggle
- Check if your API token has expired

### Permission denied on run_pipeline.sh
```bash
chmod +x run_pipeline.sh
```