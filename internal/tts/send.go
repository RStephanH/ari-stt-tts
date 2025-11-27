package tts

import (
	"context"
	// "io"
	// "os"

	"github.com/charmbracelet/log"
	apiClient "github.com/deepgram/deepgram-go-sdk/pkg/api/speak/v1/rest"
	client "github.com/deepgram/deepgram-go-sdk/pkg/client/speak/v1/rest"
	// interfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces"
	apiSpeakResponseInterfaces "github.com/deepgram/deepgram-go-sdk/pkg/api/speak/v1/rest/interfaces"
	interfaces "github.com/deepgram/deepgram-go-sdk/pkg/client/interfaces/v1"
)

func GetDgTTS(ctx context.Context, text string, raw *interfaces.RawResponse) (*apiSpeakResponseInterfaces.SpeakResponse, error) {
	speakOptions := &interfaces.SpeakOptions{
		Model:      "aura-2-thalia-en",
		Encoding:   "linear16",
		Container:  "none",
		SampleRate: 16000,
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
