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
	cfg := ariutil.ConfigFromEnv()
	cl, err := ariutil.NewARIClient(cfg)
	if err != nil {
		log.Fatal("connect failed", "err", err)
	}
	log.Info("Client connected")
	defer cl.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		log.Warn("Shutdown request!")
		cancel()
	}()

	ivr.Start(ctx, cl, cfg)
}
