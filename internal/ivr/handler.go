package ivr

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"ari/internal/ai"
	"ari/internal/stt"

	"ari/internal/externalmedia"
	"ari/internal/tts"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
	apiPrerecordedInterfaces "github.com/deepgram/deepgram-go-sdk/pkg/api/prerecorded/v1/interfaces"
	apiSpeakResponseInterfaces "github.com/deepgram/deepgram-go-sdk/pkg/api/speak/v1/rest/interfaces"
	"github.com/deepgram/deepgram-go-sdk/pkg/client/interfaces"
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
	var recResBody apiPrerecordedInterfaces.PreRecordedResponse
	var speakResBody apiSpeakResponseInterfaces.SpeakResponse

	for {
		select {
		case <-mainCtx.Done():
			log.Info("Main context done, exiting DTMF handler loop", "Channel", h.ID())
			return

		default:
			DTMFHandl(mainCtx, "sound:rick-astley", client, h, secondRecord(&recFilename, &recResBody, &speakResBody)) //Second record with listen option and another message
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

func ValidateSend(filename *string, recResBody *apiPrerecordedInterfaces.PreRecordedResponse, speakResBody *apiSpeakResponseInterfaces.SpeakResponse) ChannelHandler {
	return func(ctx context.Context, h *ari.ChannelHandle) error {
		//Get the recording bite audio
		audio, err := downloadRecordingFromARI(ctx, filename)
		if err != nil {
			return err

		}
		reader := bytes.NewReader(audio)
		//Send to Deepgram STT
		err = stt.DgSendPreRecorded(ctx, reader, recResBody)
		if err != nil {
			return err
		}

		//Get the transcription result
		fmt.Printf("Request ID: %s\n", recResBody.RequestID)
		if recResBody.Results != nil &&
			len(recResBody.Results.Channels) > 0 &&
			len(recResBody.Results.Channels[0].Alternatives) > 0 {
			transcript := recResBody.Results.Channels[0].Alternatives[0].Transcript
			fmt.Println("Transcription:", transcript)

			//Gemini Part
			gemClient, err := ai.GeminiClient(ctx) //Create Gemini client
			if err != nil {
				return err
			}
			log.Info("Gemini client created")
			gemChat, err := ai.GeminiChatClient(ctx, gemClient) //Create Gemini chat session
			if err != nil {
				return err

			}
			log.Info("Gemini chat session created")

			// Send the request to Gemini
			reqResult, err := ai.SendGeminiMessage(ctx, gemChat, transcript) //Send the transcript to Gemini
			if err != nil {
				log.Error("Error sending message to Gemini", "error", err)
				return err
			}
			log.Info("Gemini response received", "response", reqResult)

			// ---------------------TTS Part---------------------
			var raw interfaces.RawResponse
			speakResponse, eror := tts.GetDgTTS(ctx, reqResult, &raw)

			if eror != nil {
				log.Error("Error in TTS:", "error", eror)
				return eror
			}
			// ---Result verification---
			log.Info("TTS format",
				"TransferEncoding", speakResponse.TransferEncoding,
				"ModelName", speakResponse.ModelName,
				"ContextType", speakResponse.ContextType,
				"Characters", speakResponse.Characters,
			)
			log.Info("Audio received", "bytes", raw.Len())
			log.Info("Calculate", "len(raw.data) % 640", raw.Len()%640)

			// ---External Media Part---

			params := externalmedia.ExternalMediaParams{
				ARIBaseURL: os.Getenv("ARI_EXTERNAL_MEDIA_BASE_URL"),
				Username:   os.Getenv("ARI_USERNAME"),
				Password:   os.Getenv("ARI_PASSWORD"),

				AppName: os.Getenv("ARI_APPLICATION_NAME"),
				HostIP:  os.Getenv("ARI_IP"),
				Port:    4002,
				Format:  "slin16",
			}
			result, err := externalmedia.CreateExternalMedia(params)
			if err != nil {
				log.Fatal("External Media creation failed", "error", err)
			}
			log.Info("Channel RTP Info", "Channel ID", result.ID)
			log.Info("Channel RTP Info", "Asterisk RTP Address", result.ChannelVars.RTPAddress)
			log.Info("Channel RTP Info", "Asterisk RTP Port:", result.ChannelVars.RTPPort)

			rtpAddr := fmt.Sprintf("%s:%s",
				result.ChannelVars.RTPAddress,
				result.ChannelVars.RTPPort,
			)
			log.Info("Connecting channel to External Media at", "RTP Address", rtpAddr)
		}

		return nil

	}
}
