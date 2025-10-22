# OEE Factory Monitor - AI Agent Instructions

## Project Overview

IoT simulator for Overall Equipment Effectiveness (OEE) monitoring. Simulates factory machines that publish production and status events to MQTT broker for downstream backend processing.

## Architecture

### System Design

- **IoT Simulator** (`iot_simulator/main.go`): Standalone Go application simulating multiple factory machines
- **Message Bus**: EMQX MQTT broker (runs via Docker Compose on port 1883)
- **Data Flow**: Simulator → EMQX → Backend (assumed, not in this repo)
  - Status events: `factory/machine/{id}/status`
  - Production events: `factory/machine/{id}/production`
- **Containerization**: Both simulator and EMQX run as Docker services orchestrated by docker-compose.yml

### Machine Simulation Model

Each machine runs in its own goroutine with a state machine:

- **States**: `running` or `stopped`
- **OEE Components Simulated**:
  - **Availability**: Random downtime (10% chance per cycle, 10-30s duration)
  - **Performance**: Slow cycles (20% chance, up to +2s delay on 3s ideal cycle)
  - **Quality**: Scrap parts (5% defect rate)

## Key Patterns

### Configuration Style

Configuration is loaded from environment variables via `.env` file (local) or Docker environment (production):

```go
// Config struct holds all simulation parameters
type Config struct {
    MQTTBrokerURL string
    IdealCycleTime time.Duration
    ScrapRate float64
    // ... etc
}
```

- **Local development**: Copy `.env.example` to `.env` and customize
- **Docker**: Environment variables set in `docker-compose.yml`
- **Loading**: Uses `godotenv` package with fallback defaults in `loadConfig()`

### MQTT Publishing Pattern

Always call `token.Wait()` after publish to ensure delivery:

```go
token := client.Publish(topic, 0, false, payload)
token.Wait()
if token.Error() != nil {
    log.Printf("ERROR: %v", token.Error())
}
```

### Event Schema

Two event types with JSON serialization:

- `StatusEvent`: `{machine_id, status, timestamp}`
- `ProductionEvent`: `{machine_id, parts_produced, parts_scrapped, timestamp}`

All timestamps use `time.Now().UTC()`.

## Development Workflows

### Running with Docker (Recommended)

```bash
# Start EMQX and simulator together
docker-compose up -d

# View simulator logs
docker-compose logs -f simulator

# Stop all services
docker-compose down
```

### Running Locally (Development)

```bash
# 1. Start EMQX broker (via Docker)
docker-compose up -d emqx

# 2. Copy and configure .env
cp .env.example .env
# Edit .env: set MQTT_BROKER_URL=tcp://localhost:1883

# 3. Run simulator
cd iot_simulator
go run main.go
```

### Dependencies

- `github.com/eclipse/paho.mqtt.golang` - MQTT client
- `github.com/joho/godotenv` - Environment variable loading
- Install with: `go mod download`

### Machine IDs

Configure via `MACHINE_IDS` env var (comma-separated). These IDs must exist in backend's `machines` table (backend repo not included here). Example: `MACHINE_IDS=1,2,3,4,5`

## Important Constraints

- **No tests**: Project currently has no test suite
- **Blocking select**: Main goroutine uses `select{}` to run indefinitely
- **Logging**: Production events not logged (too verbose); status changes ARE logged
- **Error handling**: MQTT failures logged but don't crash individual machines
- **No graceful shutdown**: Uses indefinite blocking; terminate with SIGINT/SIGTERM

## Making Changes

When modifying simulation behavior:

1. **Parameter tuning**: Update `.env` or `docker-compose.yml` environment variables (no code changes needed)
2. **Behavior changes**: Modify `simulateMachine()` state machine logic in `main.go`
3. **New parameters**: Add to `Config` struct, `loadConfig()`, `.env.example`, and `docker-compose.yml`
4. **Schema changes**: Update event structs if changing MQTT message schema (coordinate with backend)
5. **Docker changes**: Rebuild image after code changes: `docker-compose up --build`

Note: Machine goroutines share same `rand.Rand` instance - modify if true randomness needed.
