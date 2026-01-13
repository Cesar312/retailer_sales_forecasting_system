#!/usr/bin/env bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "==================================="
echo "Walmart Data Pipeline Setup Script"
echo "Project root: ${PROJECT_ROOT}"
echo "Pipeline dir: ${SCRIPT_DIR}"
echo "==================================="

# Choose compose command (Docker Desktop usually supports `docker compose`)
if docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD=(docker compose)
else
  COMPOSE_CMD=(docker-compose)
fi

# Database env (keep your defaults, but allow overrides)
export DB_HOST="${DB_HOST:-localhost}"
export DB_PORT="${DB_PORT:-5433}"
export DB_USER="${DB_USER:-walmart_user}"
export DB_PASSWORD="${DB_PASSWORD:-walmart_pass}"
export DB_NAME="${DB_NAME:-walmart_db}"

# Data + credentials (new structure)
export RAW_DATA_DIR="${RAW_DATA_DIR:-${PROJECT_ROOT}/data/raw/walmart}"
export KAGGLE_CREDENTIALS_PATH="${KAGGLE_CREDENTIALS_PATH:-${PROJECT_ROOT}/.secrets/kaggle.json}"

mkdir -p "${RAW_DATA_DIR}"

echo -e "\n${YELLOW}Checking Docker...${NC}"
if ! docker info >/dev/null 2>&1; then
  echo -e "${RED}Error: Docker is not running. Start Docker Desktop and try again.${NC}"
  exit 1
fi
echo -e "${GREEN}Docker is running.${NC}"

echo -e "\n${YELLOW}Checking Kaggle credentials...${NC}"
if [[ ! -f "${KAGGLE_CREDENTIALS_PATH}" ]]; then
  echo -e "${RED}Error: Kaggle credentials not found at:${NC} ${KAGGLE_CREDENTIALS_PATH}"
  echo "Place kaggle.json there, or set KAGGLE_CREDENTIALS_PATH to the correct file."
  exit 1
fi
echo -e "${GREEN}Kaggle credentials found.${NC}"

echo -e "\n${YELLOW}Updating Docker images...${NC}"
docker pull postgres:16-alpine >/dev/null

echo -e "\n${YELLOW}Stopping any existing PostgreSQL container...${NC}"
(
  cd "${SCRIPT_DIR}"
  "${COMPOSE_CMD[@]}" down 2>/dev/null || true
)

echo -e "\n${YELLOW}Starting PostgreSQL container...${NC}"
(
  cd "${SCRIPT_DIR}"
  "${COMPOSE_CMD[@]}" up -d
)

echo -e "\n${YELLOW}Waiting for PostgreSQL to be ready...${NC}"
until docker exec walmart_postgres pg_isready -U "${DB_USER}" -d "${DB_NAME}" >/dev/null 2>&1; do
  echo "Waiting for PostgreSQL..."
  sleep 2
done
echo -e "${GREEN}PostgreSQL is ready.${NC}"

echo -e "\n${YELLOW}Downloading Go dependencies...${NC}"
(
  cd "${SCRIPT_DIR}"
  go mod tidy
)

echo -e "\n${YELLOW}Building Go application...${NC}"
(
  cd "${SCRIPT_DIR}"
  mkdir -p bin
  go build -o bin/walmart-pipeline main.go
)
echo -e "${GREEN}Build successful.${NC}"

echo -e "\n${YELLOW}Running the data pipeline...${NC}"
(
  cd "${SCRIPT_DIR}"
  # Go reads env vars exported above
  ./bin/walmart-pipeline
)

echo -e "\n${GREEN}==================================="
echo "Pipeline completed successfully!"
echo "===================================${NC}"

echo -e "\nYou can now query the data using:"
echo "  docker exec -it walmart_postgres psql -U ${DB_USER} -d ${DB_NAME}"
