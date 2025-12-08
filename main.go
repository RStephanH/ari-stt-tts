package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"ari/internal/ariutil"
	"ari/internal/ivr"

	"github.com/charmbracelet/log"
)

func main() {

	log.Info("Loging the environment variable", "variable", os.Getenv("ARI_USERNAME"))
	if os.Getenv("ARI_USERNAME") == "" {
		log.Fatal("Failed to retrieve the environnement variable")
	}

	// Create ARI client
	cl, err := ariutil.NewARIClient()

	if err != nil {
		log.Fatal("connect failed", "err", err)
		return
	}
	log.Info("Client connected")
	defer cl.Close()

	// Context for managing shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Signal handling for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		log.Warn("Shutdown request!")
		cancel()
	}()

	ivr.Start(ctx, cl)

}
