package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
)

// Configuration loaded from environment variables
type Config struct {
	MQTTBrokerURL           string
	MQTTClientID            string
	MachineIDs              []int
	IdealCycleTime          time.Duration
	ScrapRate               float64
	DowntimeChance          float64
	DowntimeMin             time.Duration
	DowntimeMax             time.Duration
	PerformanceLossChance   float64
	PerformanceLossMaxDelay time.Duration
}

// Global config instance
var config Config

// loadConfig loads configuration from environment variables
func loadConfig() (Config, error) {
	// Load .env file if it exists (ignored in Docker, where env vars are set directly)
	_ = godotenv.Load()

	cfg := Config{
		MQTTBrokerURL: getEnv("MQTT_BROKER_URL", "tcp://localhost:1883"),
		MQTTClientID:  getEnv("MQTT_CLIENT_ID", "oee-simulator"),
	}

	// Parse machine IDs
	machineIDsStr := getEnv("MACHINE_IDS", "1,2,3")
	ids := strings.Split(machineIDsStr, ",")
	cfg.MachineIDs = make([]int, 0, len(ids))
	for _, idStr := range ids {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			return cfg, fmt.Errorf("invalid machine ID '%s': %w", idStr, err)
		}
		cfg.MachineIDs = append(cfg.MachineIDs, id)
	}

	// Parse timing values
	idealCycleTimeSec, err := strconv.Atoi(getEnv("IDEAL_CYCLE_TIME", "3"))
	if err != nil {
		return cfg, fmt.Errorf("invalid IDEAL_CYCLE_TIME: %w", err)
	}
	cfg.IdealCycleTime = time.Duration(idealCycleTimeSec) * time.Second

	cfg.ScrapRate, err = strconv.ParseFloat(getEnv("SCRAP_RATE", "0.05"), 64)
	if err != nil {
		return cfg, fmt.Errorf("invalid SCRAP_RATE: %w", err)
	}

	cfg.DowntimeChance, err = strconv.ParseFloat(getEnv("DOWNTIME_CHANCE", "0.1"), 64)
	if err != nil {
		return cfg, fmt.Errorf("invalid DOWNTIME_CHANCE: %w", err)
	}

	downtimeMinSec, err := strconv.Atoi(getEnv("DOWNTIME_MIN", "10"))
	if err != nil {
		return cfg, fmt.Errorf("invalid DOWNTIME_MIN: %w", err)
	}
	cfg.DowntimeMin = time.Duration(downtimeMinSec) * time.Second

	downtimeMaxSec, err := strconv.Atoi(getEnv("DOWNTIME_MAX", "30"))
	if err != nil {
		return cfg, fmt.Errorf("invalid DOWNTIME_MAX: %w", err)
	}
	cfg.DowntimeMax = time.Duration(downtimeMaxSec) * time.Second

	cfg.PerformanceLossChance, err = strconv.ParseFloat(getEnv("PERFORMANCE_LOSS_CHANCE", "0.20"), 64)
	if err != nil {
		return cfg, fmt.Errorf("invalid PERFORMANCE_LOSS_CHANCE: %w", err)
	}

	perfLossMaxDelaySec, err := strconv.Atoi(getEnv("PERFORMANCE_LOSS_MAX_DELAY", "2"))
	if err != nil {
		return cfg, fmt.Errorf("invalid PERFORMANCE_LOSS_MAX_DELAY: %w", err)
	}
	cfg.PerformanceLossMaxDelay = time.Duration(perfLossMaxDelaySec) * time.Second

	return cfg, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// StatusEvent represents a machine changing its operational state.
type StatusEvent struct {
	MachineID int       `json:"machine_id"`
	Status    string    `json:"status"` // e.g., "running", "stopped"
	Timestamp time.Time `json:"timestamp"`
}

// ProductionEvent represents a machine producing parts.
type ProductionEvent struct {
	MachineID     int       `json:"machine_id"`
	PartsProduced int       `json:"parts_produced"`
	PartsScrapped int       `json:"parts_scrapped"`
	Timestamp     time.Time `json:"timestamp"`
}

// connectMQTT establishes a connection to the MQTT broker.
func connectMQTT(brokerURL, clientID string) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.OnConnect = func(c mqtt.Client) {
		log.Printf("Connected to MQTT broker at %s", brokerURL)
	}
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT: %w", token.Error())
	}
	return client, nil
}

// main is the entry point. It connects to MQTT and launches machine goroutines.
func main() {
	// Load configuration from environment variables
	var err error
	config, err = loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded:")
	log.Printf("  MQTT Broker: %s", config.MQTTBrokerURL)
	log.Printf("  Machine IDs: %v", config.MachineIDs)

	// Seed the random number generator
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	// Connect to MQTT
	client, err := connectMQTT(config.MQTTBrokerURL, config.MQTTClientID)
	if err != nil {
		log.Fatalf("Fatal error: %v. Is your MQTT broker running?", err)
		os.Exit(1)
	}
	// Disconnect gracefully on exit
	defer client.Disconnect(250)

	log.Printf("Starting IoT simulator for %d machines...", len(config.MachineIDs))

	for _, id := range config.MachineIDs {
		// Launch a new goroutine for each machine.
		// Pass the MQTT client to each one.
		go simulateMachine(client, id, r)
	}

	// Block the main goroutine forever so the program doesn't exit.
	select {}
}

// simulateMachine runs an infinite loop for a single machine's lifecycle.
func simulateMachine(client mqtt.Client, machineID int, r *rand.Rand) {
	// All machines start in the "running" state
	currentState := "running"
	sendStatusEvent(client, machineID, currentState)

	for {
		if currentState == "running" {
			// --- RUNNING STATE ---

			// --- Simulate Performance Loss ---
			actualCycleTime := config.IdealCycleTime
			if r.Float64() < config.PerformanceLossChance {
				// Machine is running slow
				delay := time.Duration(r.Intn(int(config.PerformanceLossMaxDelay)))
				actualCycleTime += delay
				// Optional: log the performance loss
				// log.Printf("[Machine %d] Performance loss: +%v", machineID, delay)
			}

			// Wait for the (potentially slower) cycle time
			time.Sleep(actualCycleTime)

			// Decide if it's a good part or scrap
			partsProduced := 0
			partsScrapped := 0
			if r.Float64() < config.ScrapRate {
				partsScrapped = 1 // It's a bad part
			} else {
				partsProduced = 1 // It's a good part
			}
			sendProductionEvent(client, machineID, partsProduced, partsScrapped)

			// After a cycle, check if the machine should go down (Availability loss)
			if r.Float64() < config.DowntimeChance {
				currentState = "stopped"
				sendStatusEvent(client, machineID, currentState)
			}

		} else {
			// --- STOPPED STATE ---
			// Simulate a random downtime duration
			downtime := time.Duration(r.Intn(int(config.DowntimeMax-config.DowntimeMin)) + int(config.DowntimeMin))
			log.Printf("[Machine %d] is DOWN for %v", machineID, downtime)
			time.Sleep(downtime)

			// Time to come back online
			currentState = "running"
			sendStatusEvent(client, machineID, currentState)
		}
	}
}

// sendStatusEvent publishes a status event to MQTT.
func sendStatusEvent(client mqtt.Client, machineID int, status string) {
	topic := fmt.Sprintf("factory/machine/%d/status", machineID)
	event := StatusEvent{
		MachineID: machineID,
		Status:    status,
		Timestamp: time.Now().UTC(),
	}
	payload, _ := json.Marshal(event)

	log.Printf("[Machine %d] Publishing to %s: %s", machineID, topic, status)

	// Use QoS=1 and retained=true so EMQX will persist the latest status per topic.
	// QoS=1 ensures delivery at least once; retained=true stores the last message on the broker.
	token := client.Publish(topic, 1, true, payload)
	// We use .Wait() to make sure the message is sent before proceeding
	// For super high throughput, you could remove this and just check errors
	token.Wait()
	if token.Error() != nil {
		log.Printf("[Machine %d] ERROR publishing status: %v", machineID, token.Error())
	}
}

// sendProductionEvent publishes a production event to MQTT.
func sendProductionEvent(client mqtt.Client, machineID, produced, scrapped int) {
	topic := fmt.Sprintf("factory/machine/%d/production", machineID)
	event := ProductionEvent{
		MachineID:     machineID,
		PartsProduced: produced,
		PartsScrapped: scrapped,
		Timestamp:     time.Now().UTC(),
	}
	payload, _ := json.Marshal(event)

	// Don't log every part, it's too noisy.
	// log.Printf("[Machine %d] Publishing to %s: %d good, %d scrap", machineID, topic, payload)

	// For production events we also use QoS=1 and set retained=true so the broker keeps
	// the last production event per machine (useful for immediate consumers after restart).
	token := client.Publish(topic, 1, true, payload)
	token.Wait()
	if token.Error() != nil {
		log.Printf("[Machine %d] ERROR publishing production: %v", machineID, token.Error())
	}
}
