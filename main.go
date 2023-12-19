package main

import (
	"context"
	"logagent/config"
	"logagent/processor"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"

	streamdal "github.com/streamdal/go-sdk" // Import Streamdal SDK
)

func main() {
	cfg := config.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup Streamdal client
	streamdalClient, err := streamdal.New(&streamdal.Config{
		ServerURL:   cfg.StreamdalServer,
		ServerToken: cfg.StreamdalToken,
		ServiceName: cfg.StreamdalServiceName,
		ShutdownCtx: ctx,
	})
	if err != nil {
		log.Fatalf("Failed to initialize Streamdal client: %v", err)
	}

	// Setup log processor
	p, err := processor.New(&processor.Config{
		LogStashAddr: cfg.LogstashAddr,
		ListenAddr:   cfg.ListenAddr,
		Streamdal:    streamdalClient,
		ShutdownCtx:  ctx,
	})
	if err != nil {
		log.Fatalf("Failed to initialize log processor: %v", err)
	}

	// Capture SIGINT and SIGTERM to trigger a shutdown.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	// Start processing logs
	p.ListenForLogs()
}
