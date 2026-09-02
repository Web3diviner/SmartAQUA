package main

import (
	"log"
	"smart-fish-feeder/internal/app"
	"smart-fish-feeder/internal/config"
)

// main is the entry point for the Smart Fish Feeder Backend API
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize and start the application
	application := app.New(cfg)
	if err := application.Run(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
}
