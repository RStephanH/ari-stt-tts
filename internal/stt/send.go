package stt

import (
	"context"
	"io"

	apiInterfaces "github.com/deepgram/deepgram-go-sdk/pkg/api/prerecorded/v1/interfaces"
	interfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces"
	client "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/listen"
)

// SendToSTT
func DgSendPreRecorded(ctx context.Context, src io.Reader, resBody *apiInterfaces.PreRecordedResponse) error {
	//Deepgram-client
	c := client.NewRESTWithDefaults()

	transcriptOptions := &interfaces.PreRecordedTranscriptionOptions{
		Language:  "en-US",
		Punctuate: true,
		Diarize:   true,
	}
	err := c.DoStream(ctx, src, transcriptOptions, resBody)
	if err != nil {
		return err
	}
	return nil
}
