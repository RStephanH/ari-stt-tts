package ivr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
	apiPrerecordedInterfaces "github.com/deepgram/deepgram-go-sdk/pkg/api/prerecorded/v1/interfaces"
	apiSpeakResponseInterfaces "github.com/deepgram/deepgram-go-sdk/pkg/api/speak/v1/rest/interfaces"
)

func RecordingRequest(filename string) ChannelHandler {
	return func(ctx context.Context, ch *ari.ChannelHandle) error {
		// The default directory for recordings is /var/spool/asterisk/recording/

		rec, err := ch.Record(filename, &ari.RecordingOptions{
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
			log.Info("Context cancelled, recording stopped.", "filename", filename)

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

func ListentRecording(filename string) ChannelHandler {
	return func(ctx context.Context, ch *ari.ChannelHandle) error {
		log.Info("Playing recording", "filename", filename)
		mediaURI := fmt.Sprintf("recording:%s", filename)
		_, err := promptSound(ctx, ch, mediaURI, []string{"#"}, 1)

		return err
	}
}

func downloadRecordingFromARI(ctx context.Context, recordingName string) ([]byte, error) {
	url := fmt.Sprintf("%s/recordings/stored/%s/file", os.Getenv("ARI_URL"), recordingName)
	log.Info("GET the ressource", "URL", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)

	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(os.Getenv("ARI_USERNAME"), os.Getenv("ARI_PASSWORD"))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)

}

func firstRecord(filename string) map[string]ChannelHandler {
	return map[string]ChannelHandler{
		"1":       RecordingRequest(filename),
		"0":       StopCall,
		"default": DoNothing,
	}
}

func secondRecord(filename string,
	recResBody *apiPrerecordedInterfaces.PreRecordedResponse,
	speakResBody *apiSpeakResponseInterfaces.SpeakResponse,
	h *ari.ChannelHandle) map[string]ChannelHandler {

	return map[string]ChannelHandler{
		"1":       RecordingRequest(filename),
		"2":       ListentRecording(filename),
		"3":       ValidateSend(filename, recResBody, speakResBody, h),
		"0":       StopCall,
		"default": DoNothing,
	}

}

func thirdRecord(filename string, filenameRes string,
	recResBody *apiPrerecordedInterfaces.PreRecordedResponse,
	speakResBody *apiSpeakResponseInterfaces.SpeakResponse,
	h *ari.ChannelHandle) map[string]ChannelHandler {

	return map[string]ChannelHandler{
		"1":       RecordingRequest(filename),
		"2":       ListentRecording(filename),
		"3":       ValidateSend(filename, recResBody, speakResBody, h),
		"4":       ListentRecording(filenameRes),
		"0":       StopCall,
		"default": DoNothing,
	}

}
