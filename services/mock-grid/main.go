package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"
)

const (
	httpPort = ":8081"
)

// PriceResponse represents the current energy pricing
type PriceResponse struct {
	Timestamp    int64   `json:"timestamp"`
	PricePerKwh  float64 `json:"price_per_kwh"`
	GridLoad     string  `json:"grid_load"`      // "Low", "Medium", "High"
	EnergySource string  `json:"energy_source"`  // "Solar", "Wind", "Grid"
	Hour         int     `json:"hour"`
}

func main() {
	http.HandleFunc("/api/pricing", handlePricing)
	http.HandleFunc("/health", handleHealth)

	log.Printf("🌞 Mock Grid Service started on %s", httpPort)
	log.Println("📊 Serving sinusoidal pricing data (high at 6pm, low at 2am)")
	
	if err := http.ListenAndServe(httpPort, nil); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}

// handlePricing returns dynamic pricing based on time of day
func handlePricing(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	hour := now.Hour()
	
	// Calculate price using sinusoidal function
	// Peak at 6pm (18:00), lowest at 2am (2:00)
	// Price formula: base + amplitude * sin(phase_shift)
	basePricePerKwh := 0.25  // $0.25 base price
	amplitude := 0.15         // $0.15 swing
	
	// Phase shift: peak at hour 18 (6pm)
	// sin wave: peaks at π/2, so we shift the hour by -6 to center peak at 18
	hoursFromMidnight := float64(hour)
	radians := (hoursFromMidnight - 6) * math.Pi / 12  // Convert to radians, shift peak
	
	pricePerKwh := basePricePerKwh + amplitude * math.Sin(radians)
	
	// Determine grid load based on price
	var gridLoad string
	if pricePerKwh > 0.35 {
		gridLoad = "High"
	} else if pricePerKwh > 0.25 {
		gridLoad = "Medium"
	} else {
		gridLoad = "Low"
	}
	
	// Determine energy source (solar during day, grid at night)
	var energySource string
	if hour >= 8 && hour <= 18 {
		energySource = "Solar"
	} else if hour >= 19 && hour <= 22 {
		energySource = "Wind"
	} else {
		energySource = "Grid"
	}
	
	// If solar, reduce price slightly
	if energySource == "Solar" {
		pricePerKwh *= 0.9
	}
	
	response := PriceResponse{
		Timestamp:    now.UnixMilli(),
		PricePerKwh:  math.Round(pricePerKwh*100) / 100,  // Round to 2 decimals
		GridLoad:     gridLoad,
		EnergySource: energySource,
		Hour:         hour,
	}
	
	log.Printf("💰 [%02d:00] Price: $%.3f/kWh | Load: %s | Source: %s",
		hour, response.PricePerKwh, gridLoad, energySource)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealth returns service health status
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}
