package ivr

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"ari/internal/ai"
	"ari/internal/ariutil"
	"ari/internal/stt"
	"ari/internal/tts"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/charmbracelet/log"
)

// CallSession holds the state for a single phone call, threaded explicitly
// through each step instead of being captured in closures or passed as
// shared pointers between functions.
type CallSession struct {
	Channel     *ari.ChannelHandle
	ARIConfig   ariutil.Config
	RecFilename string
	ResFilename string
	Transcript  *stt.TranscriptionResult
}

func newCallSession(ch *ari.ChannelHandle, cfg ariutil.Config) *CallSession {
	base := fmt.Sprintf("msg_%s_%d", ch.ID(), time.Now().Unix())
	return &CallSession{
		Channel:     ch,
		ARIConfig:   cfg,
		RecFilename: base,
		ResFilename: fmt.Sprintf("%s_tts", base),
	}
}

// transcribe downloads the caller's recording and sends it to Deepgram.
func (s *CallSession) transcribe(ctx context.Context) error {
	audio, err := downloadRecordingFromARI(ctx, s.ARIConfig, s.RecFilename)
	if err != nil {
		return fmt.Errorf("download recording: %w", err)
	}

	result, err := stt.Transcribe(ctx, stt.PreRecordedRequest{
		Source:  bytes.NewReader(audio),
		Options: stt.DefaultTranscriptionOptions(),
	})
	if err != nil {
		return fmt.Errorf("transcribe: %w", err)
	}
	s.Transcript = result
	log.Info("Transcription complete", "requestID", result.RequestID, "text", result.Text)
	return nil
}

// generateReply asks Gemini for a spoken-language response to the transcript.
func (s *CallSession) generateReply(ctx context.Context) (string, error) {
	if s.Transcript == nil || s.Transcript.Text == "" {
		return "", fmt.Errorf("no transcript available")
	}

	gemClient, err := ai.GeminiClient(ctx)
	if err != nil {
		return "", fmt.Errorf("create gemini client: %w", err)
	}
	gemChat, err := ai.GeminiChatClient(ctx, gemClient)
	if err != nil {
		return "", fmt.Errorf("create gemini chat session: %w", err)
	}

	reply, err := ai.SendGeminiMessage(ctx, gemChat, buildGeminiPrompt(s.Transcript.Text))
	if err != nil {
		return "", fmt.Errorf("send gemini message: %w", err)
	}
	return reply, nil
}

func buildGeminiPrompt(transcript string) string {
	return fmt.Sprintf(
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
}

// synthesizeReply turns the reply text into a WAV file Asterisk can play back.
func (s *CallSession) synthesizeReply(ctx context.Context, text string) (string, error) {
	const audioFormat = "wav"
	const soundsDir = "/mnt/tts"
	filePath := fmt.Sprintf("%s/%s.%s", soundsDir, s.ResFilename, audioFormat)

	if _, err := tts.GetDgFileTTS(ctx, text, filePath); err != nil {
		return "", fmt.Errorf("synthesize reply: %w", err)
	}
	log.Info("TTS file created", "file", filePath)
	return filePath, nil
}

// playReply plays the synthesized reply back to the caller.
func (s *CallSession) playReply(ctx context.Context) error {
	uri := fmt.Sprintf("recording:%s", s.ResFilename)
	if _, err := promptSound(ctx, s.Channel, uri, []string{"#"}, 1); err != nil {
		return fmt.Errorf("play reply: %w", err)
	}
	return nil
}

// respond runs the full request/response cycle: transcribe the caller's
// recording, ask Gemini for a reply, synthesize it, and play it back.
func (s *CallSession) respond(ctx context.Context) error {
	waitingSong, err := s.Channel.Play("waiting-song", "sound:rick-astley")
	if err != nil {
		log.Warn("Failed to start waiting song", "error", err)
	}
	defer func() {
		if waitingSong != nil {
			if err := waitingSong.Stop(); err != nil {
				log.Warn("Error stopping waiting song", "error", err)
			}
		}
	}()

	if err := s.transcribe(ctx); err != nil {
		return err
	}
	if s.Transcript.Text == "" {
		log.Warn("Empty transcript, nothing to respond to")
		return nil
	}

	reply, err := s.generateReply(ctx)
	if err != nil {
		return err
	}
	log.Info("Gemini response received", "response", reply)

	if _, err := s.synthesizeReply(ctx, reply); err != nil {
		return err
	}
	return s.playReply(ctx)
}

// validateAndRespondHandler adapts respond to the ChannelHandler signature
// used by the DTMF action maps in record.go.
func (s *CallSession) validateAndRespondHandler() ChannelHandler {
	return func(ctx context.Context, _ *ari.ChannelHandle) error {
		return s.respond(ctx)
	}
}
