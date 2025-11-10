package ivr

import (
	"context"
	"fmt"
	"time"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
)

func RecordingRequest(ctx context.Context, ch *ari.ChannelHandle) error {

	// The default directory for recordings is /var/spool/asterisk/recording/
	filename := fmt.Sprintf("msg_%s_%d", ch.ID(), time.Now().Unix())

	rec, err := ch.Record(filename, &ari.RecordingOptions{
		Format:      "wav",
		MaxDuration: 120 * time.Second,
		MaxSilence:  5 * time.Second,
		Exists:      "overwrite",
		Beep:        true,
		Terminate:   "#"},
	)

	go func() error {
		<-ctx.Done()
		rec.Stop()
		return ctx.Err()

	}()

	if err != nil {
		log.Errorf("Failed to start recording: %v", err)
		return err
	}
	log.Info("Started recording", "filename", filename)
	chanRec := rec.Subscribe("RecordingFinished")
	<-chanRec.Events()
	log.Info("Recording finished", "filename", filename)
	log.Info("The program should stop now!")
	return nil

}

func firstRecord() map[string]ChannelHandler {
	return map[string]ChannelHandler{
		"1":       RecordingRequest,
		"2":       StopCall,
		"default": DoNothing,
	}
}
