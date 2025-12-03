# GreenLane - Pre-Demo Checklist

Use this checklist before demonstrating the project to recruiters or running the steel thread.

## Prerequisites Installed

- [ ] Docker Desktop running
- [ ] Go 1.23+ installed (`go version`)
- [ ] Rust installed (`cargo --version`)
- [ ] Python 3.10+ installed (`python3 --version`)
- [ ] Protocol Buffers compiler installed (`protoc --version`)
- [ ] Go protobuf plugins installed:
  ```bash
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  ```

## Infrastructure Setup

- [ ] Infrastructure started: `make up`
- [ ] All containers healthy: `make status`
  - [ ] greenlane-redis (Up)
  - [ ] greenlane-redpanda (Up, healthy)
  - [ ] greenlane-console (Up)
  - [ ] greenlane-timescaledb (Up, healthy)
- [ ] Redpanda Console accessible: http://localhost:8080
- [ ] TimescaleDB responding:
  ```bash
  docker exec -it greenlane-timescaledb psql -U greenlane -d greenlane -c "SELECT 1;"
  ```

## Code Generation

- [ ] Go protobuf generated: `make proto`
- [ ] Python protobuf generated: `cd simulator && ./generate_proto.sh && cd ..`
- [ ] Verify files exist:
  - [ ] `services/ingestion/proto/fleet.pb.go`
  - [ ] `services/ingestion/proto/fleet_grpc.pb.go`
  - [ ] `simulator/fleet_pb2.py`
  - [ ] `simulator/fleet_pb2_grpc.py`

## Dependencies Downloaded

- [ ] Ingestion service: `cd services/ingestion && go mod download && cd ../..`
- [ ] Mock grid service: `cd services/mock-grid && go mod download && cd ../..`
- [ ] CLI: `cd cli && go mod download && cd ../..`
- [ ] Simulator: `cd simulator && pip3 install -r requirements.txt && cd ..`
- [ ] Pricing worker: Rust dependencies will auto-download on first `cargo build`

## Services Build

- [ ] All services built: `make build-all`
- [ ] Binaries exist in `bin/`:
  - [ ] `bin/ingestion`
  - [ ] `bin/mock-grid`
  - [ ] `bin/greenlane-cli`
  - [ ] `bin/pricing-worker` (or use `cargo run`)

## Steel Thread Test (Pre-Demo Dry Run)

### Terminal 1: Mock Grid
- [ ] Started: `make dev-mock-grid`
- [ ] Output shows: `🌞 Mock Grid Service started on :8081`
- [ ] Health check passes: `curl http://localhost:8081/health`

### Terminal 2: Ingestion Service
- [ ] Started: `make dev-ingestion`
- [ ] Output shows: `✅ Connected to Redis`
- [ ] Output shows: `✅ Connected to Redpanda (Kafka)`
- [ ] Output shows: `🚀 GreenLane Ingestion Service started on :50051`

### Terminal 3: Pricing Worker
- [ ] Started: `cd services/pricing-worker && RUST_LOG=info cargo run --release`
- [ ] Output shows: `✅ Connected to TimescaleDB`
- [ ] Output shows: `✅ Subscribed to Kafka topic: fleet-events`

### Terminal 4: Live Ops CLI
- [ ] Started: `make dev-cli`
- [ ] Output shows: `✅ Connected to Redpanda`
- [ ] Banner displays correctly

### Terminal 5: Fleet Simulator
- [ ] Started: `make dev-simulator`
- [ ] Output shows: `Spawning 5 virtual cars...`
- [ ] Color-coded output appears (🟢🟡🔴)

## Data Verification

### Redis
- [ ] Cars stored in geospatial index:
  ```bash
  docker exec -it greenlane-redis redis-cli GEORADIUS fleet:locations -74.0 40.7 50 km WITHCOORD
  ```
- [ ] Car metadata stored:
  ```bash
  docker exec -it greenlane-redis redis-cli HGETALL car:CAR-001
  ```

### Redpanda
- [ ] Topic `fleet-events` exists: http://localhost:8080
- [ ] Messages visible in Redpanda Console
- [ ] JSON payload format correct

### TimescaleDB
- [ ] Data written to charging_sessions:
  ```bash
  docker exec -it greenlane-timescaledb psql -U greenlane -d greenlane -c "SELECT COUNT(*) FROM charging_sessions;"
  ```
- [ ] Recent entries visible:
  ```bash
  docker exec -it greenlane-timescaledb psql -U greenlane -d greenlane -c "SELECT * FROM charging_sessions ORDER BY time DESC LIMIT 5;"
  ```

### Live Ops CLI
- [ ] Events streaming in real-time
- [ ] Battery levels color-coded correctly
- [ ] Critical battery warnings appear (<15%)

## Demo Screenshots Prepared

- [ ] Redpanda Console showing `fleet-events` topic
- [ ] Live Ops CLI with color-coded output
- [ ] TimescaleDB query results
- [ ] docker-compose.yml showing service dependencies
- [ ] Architecture diagram (ARCHITECTURE.md)

## Documentation Review

- [ ] README.md reviewed
- [ ] SETUP.md complete
- [ ] TESTING.md walkthrough tested
- [ ] ARCHITECTURE.md explains design decisions
- [ ] PROJECT_SUMMARY.md highlights key features

## Common Issues Resolved

- [ ] All ports free (6379, 19092, 8080, 5432, 50051, 8081)
  ```bash
  lsof -i :6379
  lsof -i :19092
  lsof -i :8080
  lsof -i :5432
  lsof -i :50051
  lsof -i :8081
  ```
- [ ] No "connection refused" errors in any service
- [ ] Simulator can authenticate (correct API token)
- [ ] No "topic not found" errors in consumers

## 🎬 Demo Script (3-Minute Version)

### Introduction (30 seconds)
"This is GreenLane - a distributed system for coordinating autonomous EV charging. It handles thousands of concurrent telemetry streams using event-driven architecture."

### Live Demo (2 minutes)
1. **Show Infrastructure** (15s)
   - `make status` - all services running
   - Open Redpanda Console: "Here's our event bus"

2. **Start Simulator** (15s)
   - `make dev-simulator`
   - "5 virtual cars moving around Manhattan"
   - Point out color coding (battery levels)

3. **Show Live Ops CLI** (30s)
   - `make dev-cli`
   - "Real-time monitoring for operators"
   - Show critical battery alert

4. **Show Data Flow** (45s)
   - Redis: `GEORADIUS fleet:locations -74.0 40.7 50 km`
   - Redpanda Console: Click on event, show JSON
   - TimescaleDB: `SELECT * FROM charging_sessions ORDER BY time DESC LIMIT 3;`

5. **Explain Architecture** (15s)
   - "gRPC for ingestion, Kafka for events, Redis for geospatial"
   - "Rust worker enriches with pricing, writes to time-series DB"

### Key Points to Emphasize
- **Scalability:** Stateless services, horizontally scalable
- **Observability:** Redpanda Console, Live Ops CLI
- **Real-world patterns:** Event sourcing, CQRS, geospatial indexing
- **Polyglot:** Go, Rust, Python - right tool for the job

## 🚨 Emergency Troubleshooting

### If services won't start:
```bash
make down
make clean  # ⚠️  Deletes all data
make up
```

### If protobuf generation fails:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
make proto
```

### If simulator can't connect:
1. Check ingestion service is running: `lsof -i :50051`
2. Verify API token in `simulator/simulator.py` matches `services/ingestion/main.go`

### If Kafka errors appear:
```bash
docker exec -it greenlane-redpanda rpk topic create fleet-events
```

## Final Checklist Before Demo

- [ ] All 5 terminals running without errors
- [ ] Live Ops CLI showing real-time events
- [ ] Redpanda Console open in browser
- [ ] Browser zoom set to 100% (for screenshots)
- [ ] Terminal font size readable (14pt+)
- [ ] No sensitive data visible (localhost only)
- [ ] Laptop plugged in (high CPU usage)
- [ ] Notifications disabled
- [ ] INMIND.md, ARCHITECTURE.md, PROJECT_SUMMARY.md ready to share

---

**Good luck with your demo! 🚀**
