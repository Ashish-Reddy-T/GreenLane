package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/segmentio/kafka-go"
)

const (
	kafkaBroker = "localhost:19092"
	kafkaTopic  = "fleet-events"
	groupID     = "live-ops-cli"
)

// TelemetryEvent represents an event from the fleet
type TelemetryEvent struct {
	CarID     string  `json:"car_id"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Battery   float64 `json:"battery"`
	Velocity  float64 `json:"velocity"`
	Timestamp int64   `json:"timestamp"`
	EventType string  `json:"event_type"`
}

// Color definitions
var (
	headerColor   = color.New(color.FgCyan, color.Bold)
	carIDColor    = color.New(color.FgYellow, color.Bold)
	criticalColor = color.New(color.FgRed, color.Bold)
	warningColor  = color.New(color.FgYellow)
	normalColor   = color.New(color.FgGreen)
	timestampColor= color.New(color.FgBlue)
)

func main() {
	// Print banner
	printBanner()

	// Create Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kafkaBroker},
		Topic:          kafkaTopic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset, // Start from latest
	})
	defer reader.Close()

	log.Println("✅ Connected to Redpanda")
	log.Printf("📡 Listening to topic: %s\n", kafkaTopic)
	fmt.Println()

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\n🛑 Shutting down...")
		cancel()
	}()

	// Stats tracking
	eventCount := 0
	startTime := time.Now()

	// Consume messages
	for {
		select {
		case <-ctx.Done():
			printStats(eventCount, startTime)
			return
		default:
			message, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("Error reading message: %v", err)
				continue
			}

			eventCount++

			// Parse event
			var event TelemetryEvent
			if err := json.Unmarshal(message.Value, &event); err != nil {
				log.Printf("Failed to parse event: %v", err)
				continue
			}

			// Display event with color coding
			displayEvent(&event, eventCount)
		}
	}
}

func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════════╗
║                    🚗 GreenLane Live Ops CLI                  ║
║                   Real-Time Fleet Monitoring                  ║
╚═══════════════════════════════════════════════════════════════╝
`
	headerColor.Println(banner)
}

func displayEvent(event *TelemetryEvent, eventNum int) {
	// Format timestamp
	timestamp := time.Unix(event.Timestamp/1000, 0).Format("15:04:05")
	
	// Choose color based on battery level
	var batteryColor *color.Color
	var batteryIcon string
	
	if event.Battery < 20 {
		batteryColor = criticalColor
		batteryIcon = "🔴"
	} else if event.Battery < 50 {
		batteryColor = warningColor
		batteryIcon = "🟡"
	} else {
		batteryColor = normalColor
		batteryIcon = "🟢"
	}

	// Format output
	fmt.Printf("[%s] %s ",
		timestampColor.Sprint(timestamp),
		batteryIcon,
	)

	fmt.Printf("%-10s | ",
		carIDColor.Sprint(event.CarID),
	)

	fmt.Printf("Battery: %s | ",
		batteryColor.Sprintf("%5.1f%%", event.Battery),
	)

	fmt.Printf("Location: (%7.4f, %8.4f) | ",
		event.Latitude,
		event.Longitude,
	)

	fmt.Printf("Speed: %4.1f km/h",
		event.Velocity,
	)

	// Add warning for critical battery
	if event.Battery < 15 {
		criticalColor.Printf(" ⚠️  CRITICAL BATTERY!")
	}

	fmt.Println()

	// Print stats every 10 events
	if eventNum%10 == 0 {
		fmt.Println(color.HiBlackString("─────────────────────────────────────────────────────────────────────────"))
	}
}

func printStats(eventCount int, startTime time.Time) {
	duration := time.Since(startTime)
	eventsPerSec := float64(eventCount) / duration.Seconds()

	fmt.Println("\n" + color.HiBlackString("═════════════════════════════════════════════════════════════════════════"))
	fmt.Println(headerColor.Sprint("📊 Session Statistics"))
	fmt.Println(color.HiBlackString("═════════════════════════════════════════════════════════════════════════"))
	fmt.Printf("Total Events:     %d\n", eventCount)
	fmt.Printf("Duration:         %s\n", duration.Round(time.Second))
	fmt.Printf("Events/Second:    %.2f\n", eventsPerSec)
	fmt.Println(color.HiBlackString("═════════════════════════════════════════════════════════════════════════"))
}
