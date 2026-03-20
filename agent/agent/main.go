package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"agent/base"
	"agent/plus"
)

func main() {
	// Check if this is the Plus process
	if len(os.Args) > 1 && os.Args[1] == "plus" {
		// Start Plus component
		log.Println("Starting Plus component...")
		plusInstance := plus.NewPlus()

		if err := plusInstance.Start(); err != nil {
			log.Fatalf("Failed to start Plus: %v", err)
		}

		// Wait for termination signal
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		// Stop Plus component
		if err := plusInstance.Stop(); err != nil {
			log.Fatalf("Failed to stop Plus: %v", err)
		}

		log.Println("Plus component stopped")
		return
	}

	// Otherwise, start Base component
	log.Println("Starting Base component...")
	baseInstance := base.NewBase()
	if err := baseInstance.Start(); err != nil {
		log.Fatalf("Failed to start Base: %v", err)
	}

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Stop Base component
	if err := baseInstance.Stop(); err != nil {
		log.Fatalf("Failed to stop Base: %v", err)
	}

	log.Println("Base component stopped")
}
