# Testing Guide

## Steel Thread End-to-End Test

This guide walks through testing the complete data flow through the GreenLane system.

## Prerequisites

Ensure all tools are installed (see SETUP.md):
- Go 1.23+
- Rust (latest)
- Python 3.10+
- Docker & Docker Compose
- protoc (Protocol Buffers compiler)

## Step 1: Start Infrastructure

```bash
cd /Users/AshishR_T/Desktop/temp/greenLane

# Start all infrastructure services
make up

# Verify services are healthy
make status

# Expected output:
# - greenlane-redis: Up
# - greenlane-redpanda: Up (healthy)
# - greenlane-console: Up
# - greenlane-timescaledb: Up (healthy)
```

Access Redpanda Console at http://localhost:8080 to verify it's running.

## Step 2: Generate Protobuf Code

```bash
# Install Go protobuf generators (if not already installed)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate Go protobuf code
make proto

# Generate Python protobuf code for simulator
cd simulator
chmod +x generate_proto.sh
./generate_proto.sh
cd ..
```

## Step 3: Build All Services

```bash
# Download Go dependencies
cd services/ingestion && go mod download && cd ../..
cd services/mock-grid && go mod download && cd ../..
cd cli && go mod download && cd ../..

# Build all services
make build-all
```

## Step 4: Run the Steel Thread

Open **6 terminal windows** and run the following commands in order:

### Terminal 1: Infrastructure Logs
```bash
cd /Users/AshishR_T/Desktop/temp/greenLane
make logs
```

### Terminal 2: Mock Grid Service
```bash
cd /Users/AshishR_T/Desktop/temp/greenLane
make dev-mock-grid
```

Expected output:
```
🌞 Mock Grid Service started on :8081
📊 Serving sinusoidal pricing data
```

### Terminal 3: Ingestion Service
```bash
cd /Users/AshishR_T/Desktop/temp/greenLane
make dev-ingestion
```

Expected output:
```
✅ Connected to Redis
✅ Connected to Redpanda (Kafka)
🚀 GreenLane Ingestion Service started on :50051
📡 Waiting for EV telemetry streams...
```

### Terminal 4: Pricing Worker (Rust)
```bash
cd /Users/AshishR_T/Desktop/temp/greenLane/services/pricing-worker
RUST_LOG=info cargo run --release
```

Expected output:
```
🦀 GreenLane Pricing Worker starting...
✅ Connected to TimescaleDB
✅ Subscribed to Kafka topic: fleet-events
📡 Listening for events...
```

### Terminal 5: Live Ops CLI
```bash
cd /Users/AshishR_T/Desktop/temp/greenLane
make dev-cli
```

Expected output:
```
╔═══════════════════════════════════════════════════════════════╗
║                    🚗 GreenLane Live Ops CLI                  ║
║                   Real-Time Fleet Monitoring                  ║
╚═══════════════════════════════════════════════════════════════╝
✅ Connected to Redpanda
📡 Listening to topic: fleet-events
```

### Terminal 6: Fleet Simulator
```bash
cd /Users/AshishR_T/Desktop/temp/greenLane/simulator

# Install Python dependencies (first time only)
pip3 install -r requirements.txt

# Run simulator
python3 simulator.py
```

Expected output:
```
🚗 GreenLane Fleet Simulator
============================================================
Spawning 5 virtual cars...
Server: localhost:50051
Update interval: 2s
============================================================

🟢 Car CAR-001 | Battery:  85.3% | Location: (40.7234, -73.9876) | Speed: 45.2 km/h
🟡 Car CAR-002 | Battery:  42.7% | Location: (40.7456, -73.9654) | Speed: 23.1 km/h
...
```

## Step 5: Verify Data Flow

### 5.1 Check Redpanda Console
1. Open http://localhost:8080
2. Navigate to **Topics** → **fleet-events**
3. You should see messages being produced in real-time
4. Click on a message to see the JSON payload:
```json
{
  "car_id": "CAR-001",
  "lat": 40.7234,
  "lon": -73.9876,
  "battery": 85.3,
  "velocity": 45.2,
  "timestamp": 1701532800000,
  "event_type": "telemetry"
}
```

### 5.2 Check Redis Data
```bash
docker exec -it greenlane-redis redis-cli

# Check geospatial data
GEORADIUS fleet:locations -73.9876 40.7234 10 km WITHCOORD

# Check car metadata
HGETALL car:CAR-001

# Exit Redis CLI
exit
```

### 5.3 Check TimescaleDB
```bash
docker exec -it greenlane-timescaledb psql -U greenlane -d greenlane

# Query charging sessions
SELECT 
    time, 
    car_id, 
    station_id, 
    kwh_usage, 
    price_rate 
FROM charging_sessions 
ORDER BY time DESC 
LIMIT 10;

# Count total sessions
SELECT COUNT(*) FROM charging_sessions;

# Average price by hour
SELECT 
    DATE_TRUNC('hour', time) as hour,
    AVG(price_rate) as avg_price,
    COUNT(*) as session_count
FROM charging_sessions
GROUP BY hour
ORDER BY hour DESC;

# Exit psql
\q
```

### 5.4 Check Live Ops CLI
In Terminal 5, you should see real-time color-coded output:
```
[20:15:32] 🟢 CAR-001    | Battery:  85.3% | Location: (40.7234, -73.9876) | Speed: 45.2 km/h
[20:15:32] 🟡 CAR-002    | Battery:  42.7% | Location: (40.7456, -73.9654) | Speed: 23.1 km/h
[20:15:32] 🔴 CAR-003    | Battery:  15.2% | Location: (40.7123, -73.9987) | Speed: 12.4 km/h ⚠️  CRITICAL BATTERY!
```

## Step 6: Chaos Testing

Test the system under stress:

### Test 1: Increase Car Count
Edit `simulator/simulator.py`:
```python
NUM_CARS = 50  # Increase from 5 to 50
```

Restart the simulator and verify all services handle the load.

### Test 2: Rapid Battery Drain
Edit `simulator/simulator.py` to drain batteries faster:
```python
self.battery_level = max(0.0, self.battery_level - random.uniform(1.0, 3.0))
```

Watch for critical battery warnings in the CLI.

### Test 3: Service Restart
Stop the Ingestion Service (Ctrl+C in Terminal 3) and restart it.
Verify the simulator reconnects automatically.

## Step 7: Performance Metrics

### Measure Throughput
In the Live Ops CLI, press Ctrl+C to see statistics:
```
📊 Session Statistics
Total Events:     500
Duration:         60s
Events/Second:    8.33
```

### Measure Latency
Check Ingestion Service logs for processing times.

### Check Resource Usage
```bash
docker stats greenlane-redis greenlane-redpanda greenlane-timescaledb
```

## Troubleshooting

### Simulator can't connect to gRPC
**Error:** `RPC Error: UNAVAILABLE`

**Solution:**
1. Ensure Ingestion Service is running
2. Check port 50051 is not blocked
3. Verify API token matches in both simulator and ingestion service

### No events in Redpanda Console
**Cause:** Topic not auto-created

**Solution:**
```bash
docker exec -it greenlane-redpanda rpk topic create fleet-events
```

### TimescaleDB connection failed
**Cause:** Database not initialized

**Solution:**
```bash
make down
make clean
make up
```

### Pricing Worker crashes
**Cause:** Mock Grid Service not running

**Solution:**
Start Mock Grid Service before Pricing Worker.

## Success Criteria

**Steel Thread is Complete When:**
1. Fleet Simulator sends telemetry via gRPC
2. Ingestion Service receives telemetry and authenticates with token
3. Redis stores geospatial data (verify with GEORADIUS)
4. Redpanda receives events (visible in Console UI)
5. Pricing Worker consumes events from Redpanda
6. Mock Grid Service provides pricing data
7. TimescaleDB stores charging sessions (verify with SELECT)
8. Live Ops CLI displays real-time events with color coding

---