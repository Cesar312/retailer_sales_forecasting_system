-- Walmart Sales Data Schema
CREATE TABLE IF NOT EXISTS walmart_sales (
    id SERIAL PRIMARY KEY,
    store INTEGER NOT NULL,
    date DATE NOT NULL,
    weekly_sales DECIMAL(12, 2) NOT NULL,
    holiday_flag BOOLEAN NOT NULL,
    temperature DECIMAL(6, 2),
    fuel_price DECIMAL(6, 3),
    cpi DECIMAL(12, 6),
    unemployment DECIMAL(6, 3),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for common queries
CREATE INDEX idx_walmart_store ON walmart_sales(store);
CREATE INDEX idx_walmart_date ON walmart_sales(date);
CREATE INDEX idx_walmart_holiday ON walmart_sales(holiday_flag);