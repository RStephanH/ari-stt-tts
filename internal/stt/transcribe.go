// transcribe.go

package stt

import (
	"context"

	apiinterfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/api/listen/v1/rest/interfaces"
)

// TranscriptionResult is FRED's own representation of a transcription,
// decoupled from Deepgram's specific response shape. Callers outside this
// package should use Transcribe and this type — not DgSendPreRecorded or
// Deepgram's SDK types directly — so a future provider change only touches
// this package.
type TranscriptionResult struct {
	RequestID string
	Text      string
}

// Transcribe sends audio to Deepgram and returns FRED's own
// TranscriptionResult. This is the entry point other packages (ivr, future
// session logic) should call.
func Transcribe(ctx context.Context, req PreRecordedRequest) (*TranscriptionResult, error) {
	resp, err := DgSendPreRecorded(ctx, req)
	if err != nil {
		return nil, err
	}
	return extractTranscript(resp), nil
}

func extractTranscript(resp *apiinterfaces.PreRecordedResponse) *TranscriptionResult {
	result := &TranscriptionResult{RequestID: resp.RequestID}
	if resp.Results != nil &&
		len(resp.Results.Channels) > 0 &&
		len(resp.Results.Channels[0].Alternatives) > 0 {
		result.Text = resp.Results.Channels[0].Alternatives[0].Transcript
	}
	return result
}
