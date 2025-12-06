package ivr

import (
	"bytes"
	"context"
	"fmt"
	// "os"
	"time"

	"ari/internal/ai"
	"ari/internal/stt"

	// "ari/internal/externalmedia"
	"ari/internal/tts"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
	apiPrerecordedInterfaces "github.com/deepgram/deepgram-go-sdk/pkg/api/prerecorded/v1/interfaces"
	apiSpeakResponseInterfaces "github.com/deepgram/deepgram-go-sdk/pkg/api/speak/v1/rest/interfaces"
	// "github.com/deepgram/deepgram-go-sdk/pkg/client/interfaces"
)

type ChannelHandler func(ctx context.Context, h *ari.ChannelHandle) error
type AfterRecordHandler func(ctx context.Context, h *ari.ChannelHandle, filename string) error

func Start(ctx context.Context, client ari.Client) {
	sub := client.Bus().Subscribe(nil, "StasisStart")
	defer sub.Cancel()

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
				log.Infof("Events StasisStart Type = %T", chanHandl)
				//Create a bridge mixed with StasisStart events
				// mixBridgeHandl, err := client.Bridge().Create(ari.NewKey(ari.BridgeKey, "mix_tts_bridge"),
				// 	"mixing",
				// 	"mix bridge for call channel and externalmedia channel")
				// bridgeID := fmt.Sprintf("mix_%s", chanHandl.Channel.ID)
				// mixBridgeHandl, err := client.Bridge().Create(ari.NewKey(ari.BridgeKey, bridgeID),
				// 	"mixing",
				// 	"mix bridge for call channel and externalmedia channel")
				// if err != nil {
				// 	log.Fatal("Failed to create bridge:", "error", err)
				// }
				// defer mixBridgeHandl.Delete()

				////Add the channel to the bridge
				//addChanRes := mixBridgeHandl.AddChannel(chanHandl.Channel.ID)
				//if addChanRes != nil {
				//	log.Fatal("Failed to add channel to bridge:", "error", addChanRes)
				//}
				//defer mixBridgeHandl.RemoveChannel(chanHandl.Channel.ID)

				//Call the handler in a separate goroutine
				go callHandl(ctx,
					client.Channel().Get(chanHandl.Key(ari.ChannelKey, chanHandl.Channel.ID)),
					client,
					chanHandl,
				)

			}
		}
	}
}

func callHandl(mainCtx context.Context,
	h *ari.ChannelHandle,
	client ari.Client,
	callChan *ari.StasisStart,
) {
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
	log.Info("print channel handler ", "channelHandler", h)

	recFilename := fmt.Sprintf("msg_%s_%d", h.ID(), time.Now().Unix())

	DTMFHandl(mainCtx,
		"sound:welcome-ari",
		client,
		h,
		firstRecord(&recFilename)) //First record wiht welcome message
	var recResBody apiPrerecordedInterfaces.PreRecordedResponse
	var speakResBody apiSpeakResponseInterfaces.SpeakResponse

	for {
		select {
		case <-mainCtx.Done():
			log.Info("Main context done, exiting DTMF handler loop", "Channel", h.ID())
			return

		default:
			DTMFHandl(mainCtx,
				"sound:rick-astley",
				client,
				h,
				secondRecord(&recFilename, &recResBody, &speakResBody, h)) //Second record with listen option and another message
		}
	}

}

func StopCall(ctx context.Context, h *ari.ChannelHandle) error {
	err := PlaySound(ctx, h, "sound:vm-goodbye")
	log.Info("Stopping call", "Channel", h.ID())
	h.Hangup()
	return err
}

func DoNothing(ctx context.Context, h *ari.ChannelHandle) error {
	log.Info("Doing nothing for Channel", "Channel", h.ID())
	return nil
}

func ValidateSend(filename *string,
	recResBody *apiPrerecordedInterfaces.PreRecordedResponse,
	speakResBody *apiSpeakResponseInterfaces.SpeakResponse,
	ch *ari.ChannelHandle,
) ChannelHandler {
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

			// --- Generate sound file of result ---- //
			audioFormat := "wav"
			pth := "/mnt/tts"
			filePath := fmt.Sprintf("%s/%s_tts.%s", pth, *filename, audioFormat)
			respFileName := fmt.Sprintf("%s_tts.%s", *filename, audioFormat)
			_, eror := tts.GetDgFileTTS(ctx, reqResult, filePath)

			if eror != nil {
				log.Error("Error in TTS:", "error", eror)
				return eror
			}
			log.Info("File created successfully", "file=", filePath)

			// --- play sound of the result ---//

			resUri := fmt.Sprintf("recording:%s", respFileName)
			log.Info("Print resUri", "resUri", resUri)
			plID := fmt.Sprintf("%s_ID", resUri)
			log.Info("print channel handler ", "channelHandler", ch)
			_, errResSoundPlay := ch.Play(plID, resUri)
			if errResSoundPlay != nil {
				log.Error("Error playing the result of the request", "filePath", filePath)
			}
			log.Info("Sound Played successfully", "sound", filePath)

		}

		return nil

	}
}
