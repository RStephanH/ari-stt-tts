// Package stt provides pre-recorded speech-to-text transcription via the
// Deepgram REST API, used by FRED to convert caller audio into text.
package stt

import (
	"context"
	"fmt"
	"io"

	api "github.com/deepgram/deepgram-go-sdk/v3/pkg/api/listen/v1/rest"
	apiinterfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/api/listen/v1/rest/interfaces"
	interfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces"
	client "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/listen"
)

// TranscriptionOptions holds the tunable parameters for a Deepgram
// pre-recorded transcription request.
type TranscriptionOptions struct {
	Language  string
	Punctuate bool
	Diarize   bool
}

func DefaultTranscriptionOptions() TranscriptionOptions {
	return TranscriptionOptions{
		Language:  "en-US",
		Punctuate: true,
		Diarize:   true,
	}
}

// PreRecordedRequest bundles the audio source with its transcription config.
type PreRecordedRequest struct {
	Source  io.Reader
	Options TranscriptionOptions
}

// DgSendPreRecorded sends a buffered audio stream to Deepgram's REST
// pre-recorded endpoint and returns the parsed transcription response.
func DgSendPreRecorded(ctx context.Context, req PreRecordedRequest) (*apiinterfaces.PreRecordedResponse, error) {
	restClient := client.NewRESTWithDefaults()
	dg := api.New(restClient)

	dgOpts := &interfaces.PreRecordedTranscriptionOptions{
		Language:  req.Options.Language,
		Punctuate: req.Options.Punctuate,
		Diarize:   req.Options.Diarize,
	}

	resp, err := dg.FromStream(ctx, req.Source, dgOpts)
	if err != nil {
		return nil, fmt.Errorf("stt: deepgram pre-recorded request: %w", err)
	}
	return resp, nil
}
