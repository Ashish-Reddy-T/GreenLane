-- GreenLane TimescaleDB Initialization Script
-- Creates the charging_sessions hypertable with proper indexing

-- 1. Create the standard table
CREATE TABLE IF NOT EXISTS charging_sessions (
    time        TIMESTAMPTZ       NOT NULL,
    session_id  TEXT              NOT NULL,
    station_id  TEXT              NOT NULL,
    car_id      TEXT              NOT NULL,
    kwh_usage   DOUBLE PRECISION  NOT NULL, -- Energy consumed
    price_rate  DOUBLE PRECISION  NOT NULL  -- Price at that moment
);

-- 2. Convert it to a Hypertable (Magic happens here)
-- Partitions data by time automatically for fast queries
SELECT create_hypertable('charging_sessions', 'time', if_not_exists => TRUE);

-- 3. Add an index for quick station lookups
CREATE INDEX IF NOT EXISTS idx_charging_sessions_station 
    ON charging_sessions (station_id, time DESC);

-- 4. Add an index for car lookups
CREATE INDEX IF NOT EXISTS idx_charging_sessions_car 
    ON charging_sessions (car_id, time DESC);

-- Grant permissions
GRANT ALL PRIVILEGES ON TABLE charging_sessions TO greenlane;
