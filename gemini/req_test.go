package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func TestGemini(t *testing.T) {
	ctx := context.Background()
	// Access your API key as an environment variable (see "Set up your API key" above)
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("API_KEY")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// For text-only input, use the gemini-pro model
	model := client.GenerativeModel("gemini-pro")
	resp, err := model.GenerateContent(ctx, genai.Text("写一则小笑话"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Candidates[0].Content.Parts[0])
}

func TestGeminiStream(t *testing.T) {
	ctx := context.Background()
	// Access your API key as an environment variable (see "Set up your API key" above)
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("API_KEY")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// For text-only input, use the gemini-pro model
	model := client.GenerativeModel("gemini-pro")
	iter := model.GenerateContentStream(ctx, genai.Text("写一则 100 字的笑话"))
	for {
		resp, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		// ... print resp
		jsonData, err := json.Marshal(resp)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(jsonData))
	}
}

func TestAskStream(t *testing.T) {
	messageCh, err := AskStream(AskStreamOptions{
		APIKey: os.Getenv("GEMINI_API_KEY"),
		Model:  "gemini-pro",
		Prompt: "如何使用 go 实现二分查找？",
	})
	if err != nil {
		log.Fatal(err)
	}
	for message := range messageCh {
		text := fmt.Sprintf("Text: %s", message.Text)
		fmt.Println(text)
	}
}
