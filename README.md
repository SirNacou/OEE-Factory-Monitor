# OEE-Factory-Monitor

IoT simulator for Overall Equipment Effectiveness (OEE) monitoring. Simulates factory machines publishing production and status events to an MQTT broker.

## Quick Start

### Using Docker (Recommended)

```bash
# Start EMQX broker and simulator
docker-compose up -d

# View simulator logs
docker-compose logs -f simulator

# Stop services
docker-compose down
```

### Local Development

```bash
# 1. Start EMQX broker
docker-compose up -d emqx

# 2. Configure environment
cp .env.example .env
# Edit .env to set MQTT_BROKER_URL=tcp://localhost:1883

# 3. Run simulator
cd iot_simulator
go run main.go
```

## Configuration

All settings are configured via environment variables. See `.env.example` for available options:

- `MQTT_BROKER_URL`: MQTT broker address
- `MACHINE_IDS`: Comma-separated machine IDs to simulate
- `IDEAL_CYCLE_TIME`: Ideal production cycle time (seconds)
- `SCRAP_RATE`: Defect probability (0.0-1.0)
- `DOWNTIME_CHANCE`: Probability of machine stopping (0.0-1.0)
- And more...

## Architecture

- **Simulator**: Go application in `iot_simulator/main.go`
- **MQTT Broker**: EMQX running on port 1883
- **Topics**:
  - `factory/machine/{id}/status` - Machine state changes
  - `factory/machine/{id}/production` - Production events
