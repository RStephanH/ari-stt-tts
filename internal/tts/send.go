package tts

import (
	"context"

	"github.com/charmbracelet/log"
	apiClient "github.com/deepgram/deepgram-go-sdk/pkg/api/speak/v1/rest"
	client "github.com/deepgram/deepgram-go-sdk/pkg/client/speak/v1/rest"

	apiSpeakResponseInterfaces "github.com/deepgram/deepgram-go-sdk/pkg/api/speak/v1/rest/interfaces"
	interfaces "github.com/deepgram/deepgram-go-sdk/pkg/client/interfaces/v1"
)

func GetDgRawTTS(ctx context.Context, text string, raw *interfaces.RawResponse) (*apiSpeakResponseInterfaces.SpeakResponse, error) {
	speakOptions := &interfaces.SpeakOptions{
		Model:      "aura-2-thalia-en",
		Encoding:   "linear16",
		Container:  "wav",
		SampleRate: 8000,
	}
	cl := client.NewWithDefaults()
	apiCl := apiClient.New(cl)
	res, err := apiCl.ToStream(ctx, text, speakOptions, raw)
	if err != nil {
		log.Error("Error getting TTS from Deepgram:", "error", err)
		return nil, err
	}

	return res, nil

}

func GetDgFileTTS(ctx context.Context, text string, filePath string) (*apiSpeakResponseInterfaces.SpeakResponse, error) {
	speakOptions := &interfaces.SpeakOptions{
		Model:      "aura-2-jupiter-en",
		Encoding:   "linear16",
		Container:  "wav",
		SampleRate: 8000,
	}

	cl := client.NewWithDefaults()
	apiCl := apiClient.New(cl)
	res, err := apiCl.ToSave(ctx, filePath, text, speakOptions)
	if err != nil {
		log.Error("Error getting TTS from Deepgram:", "error", err)
		return nil, err
	}
	log.Info("File created successfully")
	return res, nil
}
