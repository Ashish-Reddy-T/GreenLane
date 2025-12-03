# GreenLane Setup Guide

## Prerequisites Installation

### 1. Install Go 1.23

**macOS (using Homebrew):**
```bash
brew install go@1.23
```

**Manual Installation:**
1. Download Go 1.23 from https://go.dev/dl/
2. Install and add to PATH:
```bash
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$(go env GOPATH)/bin
```

### 2. Install Protocol Buffers Compiler

**macOS:**
```bash
brew install protobuf
```

**Verify installation:**
```bash
protoc --version  # Should show libprotoc 3.x or higher
```

### 3. Install Go Protobuf Plugins

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### 4. Install Rust (for Pricing Worker)

```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source $HOME/.cargo/env
```

### 5. Install Python 3.10+ (for Simulator)

**macOS:**
```bash
brew install python@3.10
```

### 6. Verify Docker is Running

```bash
docker --version
docker-compose --version
```

## Initial Setup

### 1. Clone and Navigate

```bash
cd /Users/AshishR_T/Desktop/temp/greenLane
```

### 2. Generate Protobuf Code

```bash
make proto
```

### 3. Start Infrastructure

```bash
make up
```

This will start:
- Redis (port 6379)
- Redpanda (port 19092)
- Redpanda Console (http://localhost:8080)
- TimescaleDB (port 5432)

### 4. Install Go Dependencies

```bash
cd services/ingestion && go mod download && cd ../..
cd services/mock-grid && go mod download && cd ../..
cd cli && go mod download && cd ../..
```

### 5. Install Python Dependencies

```bash
cd simulator
pip3 install -r requirements.txt
cd ..
```

### 6. Build All Services

```bash
make build-all
```

## Running the Steel Thread

### Terminal 1: Start Infrastructure
```bash
make up
make logs
```

### Terminal 2: Start Ingestion Service
```bash
make dev-ingestion
```

### Terminal 3: Start Mock Grid Service
```bash
make dev-mock-grid
```

### Terminal 4: Start Pricing Worker
```bash
cd services/pricing-worker
cargo run --release
```

### Terminal 5: Start Live Ops CLI
```bash
make dev-cli
```

### Terminal 6: Run Fleet Simulator
```bash
make dev-simulator
```

## Verification

1. **Redpanda Console**: Open http://localhost:8080 and check for `fleet-events` topic
2. **Redis**: Run `docker exec -it greenlane-redis redis-cli` and type `KEYS *`
3. **TimescaleDB**: Run `docker exec -it greenlane-timescaledb psql -U greenlane -d greenlane -c "SELECT COUNT(*) FROM charging_sessions;"`

## Troubleshooting

### "Go not found"
- Install Go 1.23 using instructions above
- Ensure `go` is in your PATH

### "protoc-gen-go not found"
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Docker containers won't start
```bash
make down
make clean
make up
```

### Port already in use
Check and kill processes on ports 6379, 19092, 8080, 5432, 50051

---