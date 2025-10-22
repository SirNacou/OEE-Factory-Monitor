package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/lib/pq"
)

// StatusEvent represents a machine status message
type StatusEvent struct {
	MachineID int       `json:"machine_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// ProductionEvent represents a production message
type ProductionEvent struct {
	MachineID     int       `json:"machine_id"`
	PartsProduced int       `json:"parts_produced"`
	PartsScrapped int       `json:"parts_scrapped"`
	Timestamp     time.Time `json:"timestamp"`
}

func mustEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	mqttURL := mustEnv("MQTT_BROKER_URL", "tcp://emqx:1883")
	mqttClientID := mustEnv("MQTT_CLIENT_ID", "oee-ingestor")
	pgHost := mustEnv("PG_HOST", "timescaledb")
	pgPort := mustEnv("PG_PORT", "5432")
	pgUser := mustEnv("PG_USER", "postgres")
	pgPass := mustEnv("PG_PASSWORD", "postgres")
	pgDB := mustEnv("PG_DB", "oee")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		pgHost, pgPort, pgUser, pgPass, pgDB)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttURL)
	opts.SetClientID(mqttClientID)
	opts.SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("failed to connect to mqtt: %v", token.Error())
	}
	defer client.Disconnect(250)

	// Subscribe to factory topics
	topics := []string{"factory/machine/+/status", "factory/machine/+/production"}
	for _, t := range topics {
		if token := client.Subscribe(t, 1, func(c mqtt.Client, m mqtt.Message) {
			handleMessage(db, m.Topic(), m.Payload())
		}); token.Wait() && token.Error() != nil {
			log.Fatalf("failed to subscribe to %s: %v", t, token.Error())
		}
	}

	log.Printf("Ingestor subscribed to topics, running...")
	select {}
}

func handleMessage(db *sql.DB, topic string, payload []byte) {
	// topic examples: factory/machine/1/status
	parts := strings.Split(topic, "/")
	if len(parts) < 4 {
		log.Printf("unknown topic format: %s", topic)
		return
	}
	machineID := 0
	fmt.Sscanf(parts[2], "%d", &machineID)
	typ := parts[3]

	switch typ {
	case "status":
		var e StatusEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			log.Printf("failed to unmarshal status: %v", err)
			return
		}
		if e.Timestamp.IsZero() {
			e.Timestamp = time.Now().UTC()
		}
		if _, err := db.Exec(`INSERT INTO status_events (time, machine_id, status) VALUES ($1,$2,$3)`, e.Timestamp, e.MachineID, e.Status); err != nil {
			log.Printf("failed to insert status event: %v", err)
		}
	case "production":
		var e ProductionEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			log.Printf("failed to unmarshal production: %v", err)
			return
		}
		if e.Timestamp.IsZero() {
			e.Timestamp = time.Now().UTC()
		}
		if _, err := db.Exec(`INSERT INTO production_events (time, machine_id, parts_produced, parts_scrapped) VALUES ($1,$2,$3,$4)`, e.Timestamp, e.MachineID, e.PartsProduced, e.PartsScrapped); err != nil {
			log.Printf("failed to insert production event: %v", err)
		}
	default:
		log.Printf("unhandled topic type: %s", typ)
	}
}
