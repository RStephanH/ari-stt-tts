package main

import (
	"ari/internal/tts"
	"context"
	"github.com/charmbracelet/log"
	"testing"
)

func TestTTS(t *testing.T) {
	msg := "Hello world!"
	filePath := "/mnt/tts.mp3"
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	_, err := tts.GetDgFileTTS(ctx, msg, filePath)
	if err != nil {
		log.Error("Error failed to get saved tts file", "error", err)
		t.Error(err)
	}

}
