package stt

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestDgSendPreRecorded_Integration exercises DgSendPreRecorded against the
// real Deepgram REST API. It is skipped when DEEPGRAM_API_KEY is unset or
// when running with -short.
func TestDgSendPreRecorded_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	if os.Getenv("DEEPGRAM_API_KEY") == "" {
		t.Skip("DEEPGRAM_API_KEY not set, skipping integration test")
	}

	f, err := os.Open("testdata/sample.wav")
	if err != nil {
		t.Fatalf("failed to open test fixture: %v", err)
	}
	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := DgSendPreRecorded(ctx, PreRecordedRequest{
		Source:  f,
		Options: DefaultTranscriptionOptions(),
	})
	if err != nil {
		t.Fatalf("DgSendPreRecorded failed: %v", err)
	}

	if resp.Results == nil || len(resp.Results.Channels) == 0 || len(resp.Results.Channels[0].Alternatives) == 0 {
		t.Fatal("expected at least one transcription alternative")
	}
	transcript := resp.Results.Channels[0].Alternatives[0].Transcript
	if !strings.Contains(strings.ToLower(transcript), "test") {
		t.Errorf("expected transcript to contain %q, got: %q", "test", transcript)
	}
}
