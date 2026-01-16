package ivr

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"ari/internal/ai"
	"ari/internal/stt"
	"ari/internal/tts"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
	apiPrerecordedInterfaces "github.com/deepgram/deepgram-go-sdk/pkg/api/prerecorded/v1/interfaces"
	apiSpeakResponseInterfaces "github.com/deepgram/deepgram-go-sdk/pkg/api/speak/v1/rest/interfaces"
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
				go callHandl(ctx,
					client.Channel().Get(chanHandl.Key(ari.ChannelKey, chanHandl.Channel.ID)),
					client,
				)

			}
		}
	}
}

func callHandl(mainCtx context.Context,
	h *ari.ChannelHandle,
	client ari.Client,
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
		firstRecord(recFilename), []string{"1", "0", "#"}) //First record wiht welcome message

	var recResBody apiPrerecordedInterfaces.PreRecordedResponse
	var speakResBody apiSpeakResponseInterfaces.SpeakResponse

	DTMFHandl(mainCtx,
		"sound:after_recording",
		client,
		h,
		secondRecord(recFilename, &recResBody, &speakResBody, h),
		[]string{"1", "2", "3", "0", "#"},
	) //Second record with listen option and another message

	resFilename := fmt.Sprintf("%s_tts", recFilename)

	for {
		select {
		case <-mainCtx.Done():
			log.Info("Main context done, exiting DTMF handler loop", "Channel", h.ID())
			return

		default:
			DTMFHandl(mainCtx,
				"sound:after_responding",
				client,
				h,
				thirdRecord(recFilename, resFilename, &recResBody, &speakResBody, h),
				[]string{"1", "2", "3", "4", "0", "#"},
			) //Third record with listen option both for the request and the response of the request
			//Also make another request
		}
	}

}

func StopCall(ctx context.Context, h *ari.ChannelHandle) error {
	err := PlaySound(ctx, h, "sound:ari_goodbye")
	log.Info("Stopping call", "Channel", h.ID())
	h.Hangup()
	return err
}

func DoNothing(ctx context.Context, h *ari.ChannelHandle) error {
	log.Info("Doing nothing for Channel", "Channel", h.ID())
	return nil
}

func ValidateSend(filename string,
	recResBody *apiPrerecordedInterfaces.PreRecordedResponse,
	speakResBody *apiSpeakResponseInterfaces.SpeakResponse,
	ch *ari.ChannelHandle,
) ChannelHandler {
	return func(ctx context.Context, h *ari.ChannelHandle) error {
		//Waiting music section
		waitingSong, err := ch.Play("waitingSong ID", "sound:rick-astley")

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

			// geminiPrompt := fmt.Sprintf(
			// 	"You are a voice assistant in a phone call, your name is FRED : F – Fast Response R – Reliable Communication E – Efficient Service Delivery D – Digitalized Interaction. Reply using plain spoken text only. Do not use markdown, lists, emojis, symbols, or formatting. Write short, clear sentences that sound natural when read aloud. Respond to the following user request: %s",
			// 	transcript,
			// )
			geminiPrompt := fmt.Sprintf(
				"You are FRED, a professional voice assistant handling a live phone call. "+
					"FRED stands for Fast Response, Reliable Communication, Efficient Service Delivery, and Digitalized Interaction. "+
					"You speak clearly, politely, and naturally, like a human service agent. "+
					"Use only plain spoken text suitable for audio. "+
					"Do not use markdown, lists, emojis, symbols, or formatting. "+
					"Keep responses short, calm, and easy to understand. "+
					"Focus on answering the caller's request or guiding them to the next step. "+
					"Do not mention artificial intelligence, models, or internal systems. "+
					"Respond to the caller's request: %s",
				transcript,
			)

			reqResult, err := ai.SendGeminiMessage(ctx, gemChat, geminiPrompt) //Send the transcript to Gemini
			if err != nil {
				log.Error("Error sending message to Gemini", "error", err)
				return err
			}
			log.Info("Gemini response received", "response", reqResult)

			// ---------------------TTS Part---------------------

			// --- Generate sound file of result ---- //
			audioFormat := "wav"
			pth := "/mnt/tts"
			URIFileName := fmt.Sprintf("%s_tts", filename)
			filePath := fmt.Sprintf("%s/%s_tts.%s", pth, filename, audioFormat)
			_, eror := tts.GetDgFileTTS(ctx, reqResult, filePath)

			if eror != nil {
				log.Error("Error in TTS:", "error", eror)
				return eror
			}
			log.Info("File created successfully", "file=", filePath)

			// --- play sound of the result ---//

			resUri := fmt.Sprintf("recording:%s", URIFileName)
			log.Info("Print resUri", "resUri", resUri)
			log.Info("print channel handler ", "channelHandler", ch)
			//stop waiting song
			if waitingSong != nil {
				err := waitingSong.Stop()
				if err != nil {
					log.Warn("Error stoping waiting music song", "Error", err)
				}
			}
			_, errResSoundPlay := promptSound(ctx, ch, resUri, []string{"#"}, 1)
			if errResSoundPlay != nil {
				log.Error("Error playing the result of the request", "filePath", filePath)
			}
			log.Info("Sound Played successfully", "sound", filePath)
		}

		return nil

	}
}
