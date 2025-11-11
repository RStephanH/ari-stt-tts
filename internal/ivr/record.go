package ivr

import (
	"context"
	"fmt"
	"time"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
)

func RecordingRequest(filename *string) ChannelHandler {
	return func(ctx context.Context, ch *ari.ChannelHandle) error {
		// The default directory for recordings is /var/spool/asterisk/recording/
		//
		// filename := fmt.Sprintf("msg_%s_%d", ch.ID(), time.Now().Unix())

		rec, err := ch.Record(*filename, &ari.RecordingOptions{
			Format:      "wav",
			MaxDuration: 120 * time.Second,
			MaxSilence:  5 * time.Second,
			Exists:      "overwrite",
			Beep:        true,
			Terminate:   "#"},
		)

		go func() {
			<-ctx.Done()
			rec.Stop()
			log.Info("Context cancelled, recording stopped.", "filename", *filename)

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
}

func ListentRecording(filename *string) ChannelHandler {
	return func(ctx context.Context, ch *ari.ChannelHandle) error {
		log.Info("Playing recording", "filename", *filename)
		plabackId := fmt.Sprintf("recording:%s", *filename)
		mediaURI := fmt.Sprintf("recording:%s", *filename)
		_, err := ch.Play(plabackId, mediaURI)
		return err
	}
}

func firstRecord(filename *string) map[string]ChannelHandler {
	return map[string]ChannelHandler{
		"1":       RecordingRequest(filename),
		"0":       StopCall,
		"default": DoNothing,
	}
}

func secondRecord(filename *string) map[string]ChannelHandler {
	return map[string]ChannelHandler{
		"1":       RecordingRequest(filename),
		"2":       ListentRecording(filename),
		"0":       StopCall,
		"default": DoNothing,
	}

}
