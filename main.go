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
	// "github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/client/native"
	"github.com/charmbracelet/log"
)

func main() {
	clientOptions := native.Options{
		Application:  "ai-ivr-app",
		Username:     "ari_user",
		Password:     "password",
		URL:          "http://192.168.122.113:8088/ari",
		WebsocketURL: "ws://192.168.122.113:8088/ari/events?app=ai-ivr-app&api_key=ari_user:password",
		SubscribeAll: true,
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

}
