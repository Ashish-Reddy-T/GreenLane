# GreenLane ⚡🚗

A distributed, high-frequency energy negotiation engine for autonomous EV fleets using event-driven architecture.

> **Status:** ✅ Steel Thread Prototype Complete  
> **Demo Ready:** Full end-to-end data flow implemented

## What Is This?

GreenLane solves the **"Thundering Herd" problem** for autonomous EV fleets competing for limited charging infrastructure. It demonstrates production-grade distributed systems patterns:

- **High-throughput ingestion** (5k+ events/sec)
- **Geospatial indexing** (Redis GEO commands)
- **Event-driven architecture** (Kafka/Redpanda)
- **Time-series analytics** (TimescaleDB hypertables)
- **Real-time monitoring** (Live Ops CLI)

## Quick Start

### Option 1: Automated Setup
```bash
./quickstart.sh
```

### Option 2: Manual Setup

#### Prerequisites
- Docker & Docker Compose
- Go 1.23+ ([install](https://go.dev/dl/))
- Rust ([install](https://rustup.rs/))
- Python 3.10+
- Protocol Buffers compiler (`brew install protobuf`)

#### Start Services
```bash
# 1. Start infrastructure (Redis, Redpanda, TimescaleDB)
make up

# 2. Generate protobuf code
make proto
cd simulator && ./generate_proto.sh && cd ..

# 3. Install Python dependencies
cd simulator && pip3 install -r requirements.txt && cd ..

# 4. Run the steel thread (open 5 terminals):
# Terminal 1:
make dev-mock-grid

# Terminal 2:
make dev-ingestion

# Terminal 3:
cd services/pricing-worker && RUST_LOG=info cargo run --release

# Terminal 4:
make dev-cli

# Terminal 5:
make dev-simulator
```

## What You'll See

### Live Ops CLI Output
```
╔═══════════════════════════════════════════════════════════════╗
║                    🚗 GreenLane Live Ops CLI                  ║
║                   Real-Time Fleet Monitoring                  ║
╚═══════════════════════════════════════════════════════════════╝

[20:15:32] 🟢 CAR-001    | Battery:  85.3% | Location: (40.7234, -73.9876) | Speed: 45.2 km/h
[20:15:32] 🟡 CAR-002    | Battery:  42.7% | Location: (40.7456, -73.9654) | Speed: 23.1 km/h
[20:15:32] 🔴 CAR-003    | Battery:  12.1% | Location: (40.7123, -73.9987) | Speed:  8.3 km/h ⚠️  CRITICAL BATTERY!
```

### Access Points
- **Redpanda Console**: http://localhost:8080 (view events in real-time)
- **TimescaleDB**: `postgresql://greenlane:greenlane_password@localhost:5432/greenlane`
- **Redis**: `localhost:6379`
- **Mock Grid API**: http://localhost:8081/api/pricing

## Architecture

```
┌─────────────┐  gRPC Stream   ┌──────────────┐   GEOADD    ┌───────┐
│ EV Fleet    │───────────────>│  Ingestion   │────────────>│ Redis │
│ (Simulator) │  Auth Token    │  Service(Go) │   HSET      │  GEO  │
└─────────────┘                └──────┬───────┘             └───────┘
                                      │ Emit Event
                                      ▼
                              ┌────────────────┐
                              │   Redpanda     │
                              │ (Kafka API)    │
                              └───────┬────────┘
                                      │ Consume
                    ┌─────────────────┼─────────────────┐
                    ▼                 ▼                 ▼
            ┌──────────────┐  ┌──────────────┐  ┌──────────┐
            │ Pricing      │  │ Live Ops     │  │ Future:  │
            │ Worker(Rust) │  │ CLI (Go)     │  │Dashboard │
            └──────┬───────┘  └──────────────┘  └──────────┘
                   │ HTTP GET                    
                   ▼                             
         ┌──────────────────┐                   
         │ Mock Grid        │                   
         │ Service (Go)     │                   
         └──────┬───────────┘                   
                │ Write                          
                ▼                                
         ┌──────────────────┐                   
         │  TimescaleDB     │                   
         │ (Time-Series)    │                   
         └──────────────────┘                   
```

## Project Structure

```
greenLane/
├── Makefile                    # Control center (make up, make down, make logs)
├── quickstart.sh               # Automated setup script
├── README.md                   # This file
├── SETUP.md                    # Detailed installation guide
├── TESTING.md                  # End-to-end testing walkthrough
├── ARCHITECTURE.md             # Deep dive into system design
├── PROJECT_SUMMARY.md          # Executive summary
├── deploy/
│   ├── docker-compose.yml      # All infrastructure (Redis, Redpanda, TimescaleDB)
│   └── init-db.sql             # TimescaleDB schema initialization
├── proto/
│   └── fleet.proto             # gRPC service + message definitions
├── services/
│   ├── ingestion/              # Go gRPC server (port 50051)
│   ├── pricing-worker/         # Rust Kafka consumer
│   └── mock-grid/              # Go HTTP server (port 8081)
├── simulator/                  # Python gRPC client (fleet simulation)
├── cli/                        # Live Ops monitoring tool
└── scripts/
    └── generate-proto.sh       # Protobuf code generation
```

## Verification

### Check Data Flow

```bash
# 1. Verify Redis geospatial data
docker exec -it greenlane-redis redis-cli GEORADIUS fleet:locations -73.9876 40.7234 10 km WITHCOORD

# 2. Verify Redpanda events
open http://localhost:8080

# 3. Query TimescaleDB
docker exec -it greenlane-timescaledb psql -U greenlane -d greenlane -c "SELECT * FROM charging_sessions ORDER BY time DESC LIMIT 5;"

# 4. Check service status
make status
```

## Documentation

- **[SETUP.md](SETUP.md)** - Complete installation guide
- **[TESTING.md](TESTING.md)** - End-to-end testing instructions
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - System design deep dive
- **[PROJECT_SUMMARY.md](PROJECT_SUMMARY.md)** - Executive overview

## Makefile Commands

```bash
make help           # Show all available commands
make up             # Start infrastructure
make down           # Stop all services
make logs           # View logs from all containers
make proto          # Generate protobuf code
make build-all      # Build all services
make clean          # WARNING: Remove all data
make status         # Show container status
```

## Key Engineering Concepts

- **gRPC Bidirectional Streaming** - Real-time, full-duplex communication
- **Event Sourcing** - Kafka/Redpanda as source of truth
- **Geospatial Indexing** - Redis GEO commands for proximity queries
- **Time-Series Optimization** - TimescaleDB hypertables for temporal data
- **Microservices Decoupling** - Independent services communicating via events
- **Observability** - Redpanda Console + Live Ops CLI for debugging

## Roadmap

### ~~Phase 1: Steel Thread (COMPLETE)~~
- [x] gRPC telemetry ingestion
- [x] Redis geospatial storage
- [x] Redpanda event streaming
- [x] Rust pricing worker
- [x] TimescaleDB persistence
- [x] Live Ops CLI

### Phase 2: Atomic Booking Engine
- [ ] Lua script for slot reservation
- [ ] Redis distributed locking
- [ ] Booking expiration (TTL)

### Phase 3: Real-Time Dashboard
- [ ] WebSocket hub
- [ ] React + Deck.gl frontend
- [ ] Map visualization

### Phase 4: Production Hardening
- [ ] mTLS authentication
- [ ] Prometheus metrics
- [ ] Grafana dashboards
- [ ] Kubernetes deployment

## License

MIT License

---

**Built with ⚡. Brought to you by Ashish Reddy Tummuri**  
