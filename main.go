package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/client/native"
	"github.com/CyCoreSystems/ari/v5/ext/play"
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
				go app(ctx, cl.Channel().Get(c.Key(ari.ChannelKey, c.Channel.ID)), cl)
			}
		}
	}
}

func app(ctx context.Context, h *ari.ChannelHandle, client ari.Client) {
	h.Answer()
	time.Sleep(2 * time.Second)
	defer h.Hangup()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Info("Runnign app", "Channel", h.ID())

	//Welcomming message
	welcomeMessage(ctx, h)
	handleDTMF(ctx, client, h)

	end := h.Subscribe(ari.Events.StasisEnd)
	defer end.Cancel()

	//End the app when the channel goes away
	go func() {
		<-end.Events()
		cancel()
	}()

}

func handleDTMF(ctx context.Context, client ari.Client, ch *ari.ChannelHandle) {
	// TODO add functionality to handle DTMF events with functions as parameters
	sub := client.Bus().Subscribe(nil, "ChannelDtmfReceived")
	for {
		e := <-sub.Events()
		if ev, ok := e.(*ari.ChannelDtmfReceived); ok {
			if ev.Channel.ID == ch.ID() {
				switch ev.Digit {
				case "1":
					recordingRequest(ctx, ch)
					// playSound(context.Background(), ch, "sound:rick-astley")
					// log.Info("Recording signal sound played")
				}
			}
		}
	}
}

func playSound(ctx context.Context, ch *ari.ChannelHandle, soundURI string) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := play.Play(ctx, ch, play.URI(soundURI)).Err(); err != nil {
		log.Errorf("Failed to play %s error= %v", soundURI, err)
		cancel()
	}
	log.Infof("Played %s", soundURI)
}

func welcomeMessage(ctx context.Context, ch *ari.ChannelHandle) {
	playSound(ctx, ch, "sound:welcome-ari")

}

func recordingRequest(ctx context.Context, ch *ari.ChannelHandle) {

	// The default directory for recordings is /var/spool/asterisk/recording/
	filename := fmt.Sprintf("msg_%s_%d", ch.ID(), time.Now().Unix())

	rec, err := ch.Record(filename, &ari.RecordingOptions{
		Format:      "wav",
		MaxDuration: 120 * time.Second,
		MaxSilence:  5 * time.Second,
		Exists:      "overwrite",
		Beep:        true,
		Terminate:   "#"})

	if err != nil {
		log.Errorf("Failed to start recording: %v", err)
	}
	log.Info("Started recording", "filename", filename)
	chanRec := rec.Subscribe("RecordingFinished")
	<-chanRec.Events()
	ctx.Done()

}
