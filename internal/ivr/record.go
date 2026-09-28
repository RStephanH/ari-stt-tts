package ivr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"ari/internal/ariutil"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
)

func RecordingRequest(filename string) ChannelHandler {
	return func(ctx context.Context, ch *ari.ChannelHandle) error {
		rec, err := ch.Record(
			filename, &ari.RecordingOptions{
				Format:      "wav",
				MaxDuration: 120 * time.Second,
				MaxSilence:  5 * time.Second,
				Exists:      "overwrite",
				Beep:        true,
				Terminate:   "#",
			},
		)
		if err != nil {
			log.Errorf("Failed to start recording: %v", err)
			return err
		}

		go func() {
			<-ctx.Done()
			rec.Stop()
			log.Info("Context cancelled, recording stopped.", "filename", filename)
		}()

		log.Info("Started recording", "filename", filename)
		chanRec := rec.Subscribe("RecordingFinished")
		<-chanRec.Events()
		log.Info("Recording finished", "filename", filename)
		return nil
	}
}

func ListentRecording(filename string) ChannelHandler {
	return func(ctx context.Context, ch *ari.ChannelHandle) error {
		log.Info("Playing recording", "filename", filename)
		mediaURI := fmt.Sprintf("recording:%s", filename)
		_, err := promptSound(ctx, ch, mediaURI, []string{"#"}, 1)
		return err
	}
}

func downloadRecordingFromARI(ctx context.Context, cfg ariutil.Config, recordingName string) ([]byte, error) {
	url := fmt.Sprintf("%s/recordings/stored/%s/file", cfg.URL, recordingName)
	log.Info("GET the ressource", "URL", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(cfg.Username, cfg.Password)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func firstRecord(s *CallSession) map[string]ChannelHandler {
	return map[string]ChannelHandler{
		"1":       RecordingRequest(s.RecFilename),
		"0":       StopCall,
		"default": DoNothing,
	}
}

func secondRecord(s *CallSession) map[string]ChannelHandler {
	return map[string]ChannelHandler{
		"1":       RecordingRequest(s.RecFilename),
		"2":       ListentRecording(s.RecFilename),
		"3":       s.validateAndRespondHandler(),
		"0":       StopCall,
		"default": DoNothing,
	}
}

func thirdRecord(s *CallSession) map[string]ChannelHandler {
	return map[string]ChannelHandler{
		"1":       RecordingRequest(s.RecFilename),
		"2":       ListentRecording(s.RecFilename),
		"3":       s.validateAndRespondHandler(),
		"4":       ListentRecording(s.ResFilename),
		"0":       StopCall,
		"default": DoNothing,
	}
}
