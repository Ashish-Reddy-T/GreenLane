# Getting Started Guide

## Quick Navigation

- **Want to understand the system?** → Read [ARCHITECTURE.md](ARCHITECTURE.md)
- **Ready to run it?** → Follow this guide
- **Want detailed testing?** → See [TESTING.md](TESTING.md)
- **Having issues?** → Check [CHECKLIST.md](CHECKLIST.md)
- **Need installation help?** → See [SETUP.md](SETUP.md)

---

## Fastest Start (5 Minutes)

### Prerequisites Check
```bash
# Verify all tools are installed
docker --version      # Should show Docker 20+
go version           # Should show go1.23+
cargo --version      # Should show cargo 1.70+
python3 --version    # Should show Python 3.10+
protoc --version     # Should show libprotoc 3.x+ (old syntax)
```

**Missing something?** See [SETUP.md](SETUP.md) for installation instructions.

### Automated Setup
```bash
cd /Users/AshishR_T/Desktop/temp/greenLane
./quickstart.sh
```

This will:
1. Start infrastructure (Redis, Redpanda, TimescaleDB)
2. Install Go protobuf generators
3. Generate all protobuf code
4. Install Python dependencies
5. Download Go dependencies
6. Build all services

**Time:** ~3-5 minutes depending on your internet speed.

---

## Running the Steel Thread

After `quickstart.sh` completes, open **5 terminal windows**:

### Terminal 1: Mock Grid Service
```bash
cd /Users/AshishR_T/Desktop/temp/greenLane
make dev-mock-grid
```
**Expected:** `🌞 Mock Grid Service started on :8081`

### Terminal 2: Ingestion Service
```bash
cd /Users/AshishR_T/Desktop/temp/greenLane
make dev-ingestion
```
**Expected:**
```
✅ Connected to Redis
✅ Connected to Redpanda (Kafka)
🚀 GreenLane Ingestion Service started on :50051
```

### Terminal 3: Pricing Worker
```bash
cd /Users/AshishR_T/Desktop/temp/greenLane/services/pricing-worker
RUST_LOG=info cargo run --release
```
**Expected:**
```
🦀 GreenLane Pricing Worker starting...
✅ Connected to TimescaleDB
✅ Subscribed to Kafka topic: fleet-events
```

### Terminal 4: Live Ops CLI
```bash
cd /Users/AshishR_T/Desktop/temp/greenLane
make dev-cli
```
**Expected:** Color-coded banner and "Listening to topic: fleet-events"

### Terminal 5: Fleet Simulator
```bash
cd /Users/AshishR_T/Desktop/temp/greenLane
make dev-simulator
```
**Expected:** 🟢🟡🔴 color-coded car telemetry streaming

---

## Verify It's Working

### 1. Check Live Ops CLI (Terminal 4)
You should see real-time events:
```
[20:15:32] 🟢 CAR-001    | Battery:  85.3% | Location: (40.7234, -73.9876) | Speed: 45.2 km/h
```

### 2. Check Redpanda Console
Open http://localhost:8080
- Click "Topics" → "fleet-events"
- You should see messages streaming

### 3. Check Redis
```bash
docker exec -it greenlane-redis redis-cli GEORADIUS fleet:locations -74.0 40.7 50 km WITHCOORD
```
**Expected:** List of cars with coordinates

### 4. Check TimescaleDB
```bash
docker exec -it greenlane-timescaledb psql -U greenlane -d greenlane -c "SELECT COUNT(*) FROM charging_sessions;"
```
**Expected:** Number increasing over time

---

## Common Commands

### Infrastructure
```bash
make up           # Start all Docker services
make down         # Stop all Docker services
make logs         # View logs from all containers
make status       # Show container status
make clean        # WARNING: Delete all data and stop services
```

### Development
```bash
make proto        # Generate protobuf code
make build-all    # Build all services
make test         # Run all tests
```

### Individual Services
```bash
make dev-ingestion    # Run Ingestion Service
make dev-mock-grid    # Run Mock Grid Service
make dev-cli          # Run Live Ops CLI
make dev-simulator    # Run Fleet Simulator
```

---

## What You're Looking At

### The Data Flow
```
1. Python Simulator generates car telemetry
   ↓
2. gRPC stream to Ingestion Service (with auth token)
   ↓
3. Ingestion Service writes to:
   - Redis (geospatial index)
   - Redpanda (event stream)
   ↓
4. Pricing Worker consumes from Redpanda:
   - Fetches price from Mock Grid Service
   - Writes to TimescaleDB
   ↓
5. Live Ops CLI displays real-time events
```

### Access Points
- **Redpanda Console:** http://localhost:8080
- **Mock Grid API:** http://localhost:8081/api/pricing
- **TimescaleDB:** `postgresql://greenlane:greenlane_password@localhost:5432/greenlane`
- **Redis:** `localhost:6379`

---

## Troubleshooting

### "Port already in use"
```bash
# Find and kill process on port (example: 6379)
lsof -ti:6379 | xargs kill -9

# Or stop all services
make down
```

### "Go not found" or "protoc-gen-go not found"
```bash
# Install Go protobuf generators
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure they're in PATH
export PATH=$PATH:$(go env GOPATH)/bin
```

### "Simulator can't connect"
1. Check Ingestion Service is running: `lsof -i :50051`
2. Verify API token in `simulator/simulator.py` matches `services/ingestion/main.go`

### "No events in Redpanda"
```bash
# Manually create topic
docker exec -it greenlane-redpanda rpk topic create fleet-events
```

### Services won't start
```bash
# Nuclear option: clean everything and restart
make clean  # WARNING:  Deletes all data
make up
./quickstart.sh
```

---

## Next Steps

### Explore the System
1. **Read the Architecture** - [ARCHITECTURE.md](ARCHITECTURE.md)
2. **Run Full Tests** - [TESTING.md](TESTING.md)
3. **Check Pre-Demo List** - [CHECKLIST.md](CHECKLIST.md)

### Extend the System
1. Add atomic booking logic (Lua script in Redis)
2. Implement GEORADIUS for "find nearest station"
3. Add Prometheus metrics to services
4. Build a React dashboard with Deck.gl

### Customize for Demo
1. Increase car count: Edit `simulator/simulator.py` → `NUM_CARS = 50`
2. Change pricing curve: Edit `services/mock-grid/main.go`
3. Modify battery drain: Edit `simulator/simulator.py`

---

## Resources

### Technologies Used
- **gRPC:** https://grpc.io/docs/languages/go/
- **Redis Geospatial:** https://redis.io/commands/geoadd/
- **Kafka/Redpanda:** https://docs.redpanda.com/
- **TimescaleDB:** https://docs.timescale.com/
- **Protocol Buffers:** https://protobuf.dev/

### Patterns Demonstrated
- Event Sourcing
- CQRS (Command Query Responsibility Segregation)
- Microservices Architecture
- Geospatial Indexing
- Time-Series Optimization
- gRPC Streaming
- Middleware/Interceptors

---

## Help

**Stuck?** Check these in order:
1. [CHECKLIST.md](CHECKLIST.md) - Pre-demo verification
2. [TESTING.md](TESTING.md) - Detailed testing guide
3. [SETUP.md](SETUP.md) - Installation help
4. [ARCHITECTURE.md](ARCHITECTURE.md) - System design

**Still stuck?** Review logs:
```bash
make logs                  # All services
make logs-redis           # Redis only
make logs-redpanda        # Redpanda only
make logs-timescale       # TimescaleDB only
```

---

## Success Criteria

You'll know it's working when:
- Live Ops CLI shows color-coded events
- Redpanda Console shows `fleet-events` topic with messages
- Redis has cars in geospatial index
- TimescaleDB has rows in `charging_sessions` table
- All 5 terminals running without errors

---

**Run `./quickstart.sh` to start building NOW!**
