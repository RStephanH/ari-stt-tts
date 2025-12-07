package ai

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	"google.golang.org/genai"
)

func GeminiClient(ctx context.Context) (*genai.Client, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func GeminiChatClient(ctx context.Context, client *genai.Client) (*genai.Chat, error) {

	chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, nil)

	if err != nil {
		return nil, err
	}
	return chat, nil
}

func SendGeminiMessage(ctx context.Context, chat *genai.Chat, message string) (string, error) {

	result, err := chat.SendMessage(ctx, genai.Part{Text: message})
	if err != nil {
		return "", err
	}
	log.Info("Gemini response:", "response", result.Text())
	return result.Text(), nil

}
