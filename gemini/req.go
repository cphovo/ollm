package gemini

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type AskStreamOptions struct {
	APIKey string
	Model  string
	Prompt string
}

type Message struct {
	Event string
	Text  string
	Error error
}

func AskStream(options AskStreamOptions) (<-chan Message, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(options.APIKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	model := client.GenerativeModel(options.Model)
	iter := model.GenerateContentStream(ctx, genai.Text(options.Prompt))

	messageChan := make(chan Message)

	go func() {
		defer close(messageChan)
		for {
			resp, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				messageChan <- Message{Error: err, Event: "error"}
				return
			}

			text := resp.Candidates[0].Content.Parts[0]
			messageChan <- Message{Text: fmt.Sprint(text), Event: "message"}
		}
	}()

	return messageChan, nil
}
