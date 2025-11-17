package ivr

import (
	"ari/internal/stt"
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
	apiInterfaces "github.com/deepgram/deepgram-go-sdk/pkg/api/prerecorded/v1/interfaces"
)

type ChannelHandler func(ctx context.Context, h *ari.ChannelHandle) error
type AfterRecordHandler func(ctx context.Context, h *ari.ChannelHandle, filename string) error

func Start(ctx context.Context, client ari.Client) {
	sub := client.Bus().Subscribe(nil, "StasisStart")
	defer sub.Cancel()

	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()

	for {
		select {
		case <-ctx.Done():
			log.Warn("context cancelled, exiting ...")
			return

		case evt, ok := <-sub.Events():
			if !ok {
				log.Warn("event channel closed")
				return
			}

			if chanHandl, ok := evt.(*ari.StasisStart); ok {
				log.Infof("c Type = %T", chanHandl)
				go callHandl(ctx, subCtx, subCancel, client.Channel().Get(chanHandl.Key(ari.ChannelKey, chanHandl.Channel.ID)), client)

			}
		}
	}
}

func callHandl(mainCtx context.Context, subCtx context.Context, subCancel context.CancelFunc, h *ari.ChannelHandle, client ari.Client) {
	h.Answer()
	time.Sleep(2 * time.Second)
	defer h.Hangup()

	mainCtx, cancel := context.WithCancel(mainCtx)
	defer cancel()

	log.Info("Runnign app", "Channel", h.ID())

	end := h.Subscribe(ari.Events.StasisEnd)
	defer end.Cancel()

	//End the app when the channel goes away
	go func() {
		<-end.Events()
		cancel()
	}()

	recFilename := fmt.Sprintf("msg_%s_%d", h.ID(), time.Now().Unix())
	DTMFHandl(mainCtx, "sound:welcome-ari", client, h, firstRecord(&recFilename)) //First record wiht welcome message
	var resBody apiInterfaces.PreRecordedResponse

	for {
		select {
		case <-mainCtx.Done():
			log.Info("Main context done, exiting DTMF handler loop", "Channel", h.ID())
			return

		default:
			DTMFHandl(mainCtx, "sound:rick-astley", client, h, secondRecord(&recFilename, &resBody)) //Second record with listen option and another message
		}
	}

}

func StopCall(ctx context.Context, h *ari.ChannelHandle) error {
	err := playSound(ctx, h, "sound:vm-goodbye")
	log.Info("Stopping call", "Channel", h.ID())
	h.Hangup()
	return err
}
func DoNothing(ctx context.Context, h *ari.ChannelHandle) error {

	log.Info("Doing nothing for Channel", "Channel", h.ID())
	return nil
}
func ValidateSend(filename *string, resBody *apiInterfaces.PreRecordedResponse) ChannelHandler {
	return func(ctx context.Context, h *ari.ChannelHandle) error {
		//Get the recording bite audio
		audio, err := downloadRecordingFromARI(ctx, filename)
		if err != nil {
			return err

		}
		reader := bytes.NewReader(audio)
		//Send to Deepgram STT
		err = stt.DgSendPreRecorded(ctx, reader, resBody)
		if err != nil {
			return err
		}
		fmt.Printf("Request ID: %s\n", resBody.RequestID)
		if resBody.Results != nil &&
			len(resBody.Results.Channels) > 0 &&
			len(resBody.Results.Channels[0].Alternatives) > 0 {
			transcript := resBody.Results.Channels[0].Alternatives[0].Transcript
			fmt.Println("Transcription:", transcript)
		}
		return nil

	}
}
