package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	// "os"
	// "os/signal"
	// "syscall"
	//
	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/client/native"
	"github.com/charmbracelet/log"
)

func main() {
	// Create ARI client
	// TODO: Move credentials to config or environment variables
	clientOptions := native.Options{
		Application:  "app",
		Username:     "ari_user",
		Password:     "password",
		URL:          "http://192.168.122.113:8088/ari",
		WebsocketURL: "ws://192.168.122.113:8088/ari/events?app=app&api_key=ari_user:password",
	}

	cl, err := native.Connect(&clientOptions)
	if err != nil {
		log.Fatal("connect failed", "err", err, "url", clientOptions.URL)
		return
	}
	if !cl.Connected() {
		log.Fatal("not connected", "url", clientOptions.URL)
		return
	}
	log.Info("connected", "url", clientOptions.URL)
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

	eventCh := cl.Bus().Subscribe(nil, "StasisStart", "StasisEnd")
	defer eventCh.Cancel()
	log.Info("Client ARI started, waiting for StasisStart and StasisEnd event ...", "LISTEN_EVENT", "TRUE")

	for {
		select {
		case <-ctx.Done():
			log.Warn("context cancelled, exiting ...")
			return

		case evt, ok := <-eventCh.Events():
			if !ok {
				log.Warn("event channel closed")
				return
			}
			log.Info("New event", "Type", evt.GetType(), "Application", evt.GetApplication())

			if evt.GetType() == "StasisStart" {
				c := evt.(*ari.StasisStart)
				go app(ctx, cl.Channel().Get(c.Key(ari.ChannelKey, c.Channel.ID)))
			}
		}
	}
}

func app(ctx context.Context, h *ari.ChannelHandle) {
	h.Answer()
	defer h.Hangup()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Info("Runnign app", "Channel", h.ID())
	end := h.Subscribe(ari.Events.StasisEnd)
	defer end.Cancel()

	//End the app when the channel goes away
	go func() {
		<-end.Events()
		cancel()
	}()

	// if err := pl
	log.Info("Something happen", "Action", true)
}
