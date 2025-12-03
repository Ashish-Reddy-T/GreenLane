# Architecture

## System Overview

GreenLane is a distributed, event-driven system for managing EV fleet charging coordination. <br> 
It handles high-frequency telemetry ingestion, geospatial indexing, atomic booking operations, and real-time pricing optimization.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        EV Fleet (Clients)                       │
│                     Python Simulator (Testing)                  │
└───────────────────────────┬─────────────────────────────────────┘
                            │ gRPC Bidirectional Stream
                            │ (CarStatus → BookingResponse)
                            │ Auth: x-api-token metadata
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Ingestion Service (Go)                       │
│  ┌─────────────┐    ┌──────────────┐    ┌──────────────┐        │
│  │ gRPC Server │───▶│ Auth         │───▶│ Stream       │        │
│  │ :50051      │    │ Interceptor  │    │ Handler      │        │
│  └─────────────┘    └──────────────┘    └──────┬───────┘        │
│                                                 │               │
│                     ┌───────────────────────────┼───────┐       │
│                     ▼                           ▼       │       │
│            ┌─────────────────┐        ┌──────────────┐ │        │
│            │ Redis Writer    │        │ Kafka Writer │ │        │
│            │ (GEOADD/HSET)   │        │ (Emit Event) │ │        │
│            └────────┬────────┘        └──────┬───────┘ │        │
└─────────────────────┼────────────────────────┼─────────┼───────┘
                      │                        │         │
                      ▼                        ▼         │
           ┌──────────────────────┐  ┌──────────────────┴──────┐
           │   Redis (Alpine)     │  │  Redpanda (Kafka API)   │
           │   Geospatial Store   │  │   Event Streaming       │
           │   :6379              │  │   :19092 (Kafka)        │
           │                      │  │   :8080 (Console UI)    │
           │ • fleet:locations    │  │                         │
           │   (GEOADD)           │  │ Topic: fleet-events     │
           │ • car:{id}           │  │                         │
           │   (HSET metadata)    │  │ Partitions: Auto        │
           └──────────────────────┘  └──────┬──────────────────┘
                                             │
                      ┌──────────────────────┼──────────────────┐
                      │                      │                  │
                      ▼                      ▼                  ▼
         ┌────────────────────┐  ┌─────────────────┐  ┌────────────────┐
         │ Pricing Worker     │  │ Live Ops CLI    │  │ (Future)       │
         │ (Rust)             │  │ (Go)            │  │ WebSocket Hub  │
         │                    │  │                 │  │ Dashboard      │
         │ Kafka Consumer     │  │ Kafka Consumer  │  └────────────────┘
         │ - Read events      │  │ - Tail events   │
         │ - Fetch pricing    │  │ - Display       │
         │ - Write TimescaleDB│  │   real-time     │
         └──────┬─────────────┘  └─────────────────┘
                │
                │ HTTP GET
                ▼
    ┌────────────────────────┐
    │ Mock Grid Service (Go) │
    │ :8081                  │
    │                        │
    │ /api/pricing           │
    │ - Sinusoidal pricing   │
    │ - Peak: 6pm            │
    │ - Low: 2am             │
    └────────────────────────┘
                │
                │ Writes charging_sessions
                ▼
    ┌────────────────────────────────────┐
    │   TimescaleDB (PostgreSQL + ext)   │
    │   :5432                            │
    │                                    │
    │   Table: charging_sessions         │
    │   - time (Hypertable partition)    │
    │   - session_id, station_id         │
    │   - car_id, kwh_usage, price_rate  │
    │                                    │
    │   Indexes:                         │
    │   - (station_id, time DESC)        │
    │   - (car_id, time DESC)            │
    └────────────────────────────────────┘
```

## Component Details

### 1. Ingestion Service (Go)
**Purpose:** High-throughput telemetry ingestion with gRPC streaming

**Key Features:**
- **Bidirectional gRPC streaming:** Cars send CarStatus, server responds with BookingResponse
- **Authentication:** Unary and Stream interceptors validate `x-api-token` metadata
- **Geospatial indexing:** Uses Redis GEOADD to store car locations
- **Event emission:** Publishes telemetry to Redpanda for downstream consumption
- **Concurrency:** Goroutines handle multiple streams simultaneously

**Technology Choices:**
- **Go:** Low-latency, excellent concurrency model (goroutines)
- **gRPC:** Efficient binary protocol, bidirectional streaming
- **Redis client:** `go-redis/redis/v8` for geospatial operations
- **Kafka client:** `segmentio/kafka-go` for event production

**Critical Code Paths:**
```go
StreamTelemetry() -> Recv() -> writeToRedis() + emitToKafka() -> Send()
```

**Authentication Flow:**
1. Client includes metadata: `x-api-token: greenlane-secret-token`
2. `authStreamInterceptor` validates token before handler invocation
3. Invalid token returns `codes.Unauthenticated`

### 2. Redis (Alpine)
**Purpose:** In-memory geospatial indexing and atomic locking

**Data Structures:**
- **`fleet:locations` (Geo):** Stores car coordinates using GEOADD
  ```
  GEOADD fleet:locations -73.9876 40.7234 CAR-001
  ```
- **`car:{id}` (Hash):** Stores metadata (battery, velocity, timestamp)
  ```
  HSET car:CAR-001 battery 85.3 velocity 45.2 timestamp 1701532800000
  ```

**Future Enhancement (Phase 2):**
Lua script for atomic booking:
```lua
-- Check capacity and decrement atomically
local capacity = redis.call('GET', 'station:' .. station_id .. ':capacity')
if tonumber(capacity) > 0 then
    redis.call('DECR', 'station:' .. station_id .. ':capacity')
    redis.call('SETEX', 'booking:' .. booking_id, 1200, car_id)  -- 20min TTL
    return 1  -- Success
end
return 0  -- No capacity
```

### 3. Redpanda (Kafka-Compatible)
**Purpose:** Event streaming backbone for decoupled microservices

**Configuration:**
- **Topic:** `fleet-events`
- **Partitions:** Auto-created (default 1, can scale)
- **Retention:** Default 7 days
- **Compression:** Snappy (configured in Ingestion Service)

**Event Schema (JSON):**
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

**Why Redpanda over Kafka:**
- Simpler deployment (single binary, no ZooKeeper)
- Lower resource usage
- Kafka API compatible (drop-in replacement)
- Built-in Console UI

### 4. Pricing Worker (Rust)
**Purpose:** Kafka consumer that enriches events with pricing and persists to TimescaleDB

**Pipeline:**
```
Kafka Consumer → Parse Event → Fetch Price (HTTP) → Write TimescaleDB
```

**Technology Choices:**
- **Rust:** Memory safety, zero-cost abstractions, high performance
- **rdkafka:** Native Kafka client (librdkafka bindings)
- **tokio:** Async runtime for I/O operations
- **tokio-postgres:** Async PostgreSQL driver

**Critical Operations:**
1. **Consume:** Read from `fleet-events` topic
2. **Enrich:** HTTP GET to Mock Grid Service for current pricing
3. **Transform:** Calculate `kwh_usage`, generate `session_id`
4. **Persist:** INSERT into TimescaleDB `charging_sessions` table

**Error Handling:**
- Kafka errors: Retry with backoff (1s sleep)
- HTTP timeouts: 5s timeout, log error, skip event
- DB errors: Log error, continue (at-least-once delivery)

### 5. Mock Grid Service (Go)
**Purpose:** Simulate time-of-use pricing from electricity grid

**Pricing Formula:**
```go
basePricePerKwh := 0.25  // $0.25
amplitude := 0.15         // $0.15 swing
radians := (hour - 6) * π / 12
pricePerKwh := base + amplitude * sin(radians)
```

**Pricing Curve:**
```
$0.40 |        ╱╲
      |       ╱  ╲
$0.30 |      ╱    ╲      Peak at 6pm
      |     ╱      ╲
$0.20 |____╱        ╲____  Low at 2am
      |                  
      0  6  12  18  24 (hour)
```

**Energy Source Logic:**
- 8am-6pm: Solar (reduce price by 10%)
- 7pm-10pm: Wind
- 11pm-7am: Grid

**Endpoint:**
- `GET /api/pricing` → Returns JSON with `price_per_kwh`, `grid_load`, `energy_source`

### 6. TimescaleDB (PostgreSQL Extension)
**Purpose:** Time-series database optimized for temporal queries

**Hypertable Magic:**
```sql
-- Standard PostgreSQL table
CREATE TABLE charging_sessions (...);

-- Convert to Hypertable (automatic time-based partitioning)
SELECT create_hypertable('charging_sessions', 'time');
```

**Benefits:**
- Automatic partitioning by time (weekly chunks)
- Fast range queries on time column
- Compression for old data
- Continuous aggregates (future: hourly stats)

**Query Patterns:**
```sql
-- Recent sessions for a car
SELECT * FROM charging_sessions 
WHERE car_id = 'CAR-001' 
ORDER BY time DESC LIMIT 10;

-- Average price by hour
SELECT DATE_TRUNC('hour', time) as hour, AVG(price_rate)
FROM charging_sessions
GROUP BY hour
ORDER BY hour DESC;

-- Station utilization
SELECT station_id, COUNT(*), SUM(kwh_usage)
FROM charging_sessions
WHERE time > NOW() - INTERVAL '1 day'
GROUP BY station_id;
```

### 7. Live Ops CLI (Go)
**Purpose:** Real-time event monitoring for operators

**Features:**
- Kafka consumer (group: `live-ops-cli`)
- Color-coded battery levels: 🟢 >50%, 🟡 20-50%, 🔴 <20%
- Critical battery alerts (<15%)
- Session statistics (events/sec, duration)

**UI Design:**
```
[20:15:32] 🟢 CAR-001    | Battery:  85.3% | Location: (40.7234, -73.9876) | Speed: 45.2 km/h
[20:15:32] 🔴 CAR-003    | Battery:  12.1% | Location: (40.7123, -73.9987) | Speed:  8.3 km/h ⚠️  CRITICAL BATTERY!
```

**Technology:**
- `fatih/color` package for terminal colors
- `segmentio/kafka-go` for consumption
- Graceful shutdown on Ctrl+C

### 8. Fleet Simulator (Python)
**Purpose:** Generate synthetic telemetry for testing

**Simulation Logic:**
- Random walk within Manhattan bounding box
- Battery drains at 0.1-0.5% per update
- Velocity randomized (0-60 km/h)
- Updates every 2 seconds

**gRPC Client:**
```python
# Create bidirectional stream
stub.StreamTelemetry(request_generator(), metadata=[("x-api-token", TOKEN)])
```

**Parallelization:**
- Uses `ThreadPoolExecutor` to simulate multiple cars concurrently
- Each car runs in its own thread with independent stream

## Data Flow: Steel Thread

**Step-by-Step:**

1. **Simulator** generates CarStatus (Lat, Lon, Battery, Velocity)
2. **gRPC Stream** sends to Ingestion Service with x-api-token
3. **Ingestion Service:**
   - Validates token (interceptor)
   - Writes to Redis: `GEOADD fleet:locations ...`
   - Writes to Redis: `HSET car:{id} ...`
   - Publishes to Redpanda: `fleet-events` topic
4. **Redpanda** stores event and fans out to consumers
5. **Pricing Worker:**
   - Consumes event from Redpanda
   - Fetches price from Mock Grid Service (HTTP)
   - Writes to TimescaleDB: `INSERT INTO charging_sessions`
6. **Live Ops CLI:**
   - Consumes event from Redpanda
   - Displays color-coded real-time output
7. **Redpanda Console:**
   - Web UI visualizes all events for debugging

## Scalability Considerations

### Horizontal Scaling
- **Ingestion Service:** Stateless, can run multiple instances behind load balancer
- **Pricing Worker:** Kafka consumer group scales with partitions
- **Redis:** Redis Cluster for sharding (future)
- **Redpanda:** Add brokers and increase partitions

### Vertical Scaling
- **Ingestion Service:** Increase goroutine pool size
- **Redis:** Increase memory allocation
- **TimescaleDB:** Increase chunk size for hypertable

### Bottlenecks
1. **Redis single-threaded:** Use Redis Cluster or Dragonfly
2. **Ingestion Service network I/O:** Add more instances
3. **TimescaleDB writes:** Batch inserts, increase work_mem

## Monitoring & Observability

### Key Metrics
- **Ingestion Service:** Active streams, events/sec, Redis latency, Kafka latency
- **Pricing Worker:** Lag (Kafka offset), HTTP request latency, DB write latency
- **Redis:** Memory usage, GEORADIUS query time, keyspace hits/misses
- **Redpanda:** Partition lag, throughput, consumer group lag
- **TimescaleDB:** Write throughput, chunk compression ratio, query latency

### Instrumentation (Future)
- Prometheus metrics exporter in Go services
- Grafana dashboards for all metrics
- Distributed tracing with Jaeger (OpenTelemetry)

## Security

### Current Implementation (POC)
- **API Token:** Simple metadata token (`x-api-token`)
- **No TLS:** Services communicate over plaintext

### Production Hardening
- **mTLS:** Mutual TLS for service-to-service communication
- **OAuth2:** Replace simple token with JWT
- **Network policies:** Kubernetes NetworkPolicy or Istio
- **Secret management:** Vault or Kubernetes Secrets
- **Rate limiting:** Per-client token bucket in Ingestion Service

## Deployment

### Local Development
```bash
make up      # Docker Compose
make proto   # Generate code
make build-all
# Run services in separate terminals
```

### Kubernetes (Future)
```
ingestion-deployment (3 replicas)
↓ ClusterIP Service
↓ Ingress (TLS)
↓ Load Balancer
```

### CI/CD (Planned)
- GitHub Actions
- golangci-lint for Go
- clippy for Rust
- pytest for Python
- Docker image builds
- Helm chart deployment

## Future Enhancements

### Phase 2: Atomic Booking
- Lua script in Redis for slot decrement
- Booking expiration (20min TTL)
- GEORADIUS to find nearby stations

### Phase 3: WebSocket Layer
- Go WebSocket hub
- Push events to web dashboard
- Real-time car visualization with Deck.gl

### Phase 4: Machine Learning
- Battery drain prediction
- Optimal charging time recommendation
- Dynamic pricing based on demand

### Phase 5: Multi-Region
- CockroachDB instead of TimescaleDB
- Multi-region Redpanda clusters
- CDN for dashboard
