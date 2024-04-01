package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cphovo/ollm/gemini"
	"github.com/cphovo/ollm/sydney"
	"github.com/cphovo/ollm/util"
	"github.com/gin-gonic/gin"
)

type GeminiChatStreamRequest struct {
	Text   string `json:"text"`
	APIKey string `json:"apiKey"`
	Model  string `json:"model"`
}

type GeminiOpenAIMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type GeminiOpenAIChatCompletionRequest struct {
	Model      string                `json:"model"`
	Messages   []GeminiOpenAIMessage `json:"messages"`
	Stream     bool                  `json:"stream"`
	ToolChoice *interface{}          `json:"tool_choice"`
	APIKey     string                `json:"refreshToken"`
}

func GeminiStreamChatHandler(c *gin.Context) {
	var request GeminiChatStreamRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Text is required"})
		return
	}

	model := util.Ternary(request.Model == "", "gemini-pro", request.Model)
	apiKey := util.Ternary(request.APIKey == "", DefaultAPIKey, request.APIKey)

	messageCh, err := gemini.AskStream(gemini.AskStreamOptions{
		APIKey: apiKey,
		Model:  model,
		Prompt: request.Text,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating conversation: " + err.Error()})
		return
	}
	c.Stream(func(w io.Writer) bool {
		for message := range messageCh {
			encoded, _ := json.Marshal(message.Text)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", message.Event, encoded)
			c.Writer.Flush()
		}
		return false
	})
}

func GeminiCompleteChatHandler(c *gin.Context) {
	var request GeminiOpenAIChatCompletionRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(request.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Messages is required"})
		return
	}

	model := util.Ternary(request.Model == "", "gemini", request.Model)
	apiKey := util.Ternary(request.APIKey == "", DefaultAPIKey, request.APIKey)

	// 将 OpenAI 格式的消息转换成 Kimi 格式的消息
	text := geminiMessagesPrepare(request.Messages)
	options := gemini.AskStreamOptions{
		APIKey: apiKey,
		Model:  model,
		Prompt: text,
	}

	if !request.Stream {
		reply, err := gemini.Ask(options)
		errored := false
		if err != nil {
			reply = err.Error()
			errored = true
		}
		c.JSON(http.StatusOK, sydney.NewOpenAIChatCompletion(
			strings.ToUpper(model),
			reply,
			util.Ternary(errored, sydney.FinishReasonLength, sydney.FinishReasonStop),
		))
		return
	}

	messageCh, err := gemini.AskStream(options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating conversation: " + err.Error()})
		return
	}

	c.Stream(func(w io.Writer) bool {
		errored := false

		for message := range messageCh {
			var delta string

			switch message.Event {
			case "message":
				delta = message.Text
			case "error":
				errored = true
				delta = fmt.Sprintf("`Error: %s`", message.Text)
			default:
				continue
			}

			chunk := sydney.NewOpenAIChatCompletionChunk(strings.ToUpper(model), delta, nil)
			encoded, err := json.Marshal(chunk)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", encoded)
			c.Writer.Flush()
		}

		chunk := sydney.NewOpenAIChatCompletionChunk(strings.ToUpper(model), "", util.Ternary(errored, &sydney.FinishReasonLength, &sydney.FinishReasonStop))
		encoded, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n", encoded)
		c.Writer.Flush()

		return false
	})
}

// TODO: 暂时和 KIMI 一样，后续修改
func geminiMessagesPrepare(messages []GeminiOpenAIMessage) string {
	var contentBuilder strings.Builder

	for _, message := range messages {
		switch content := message.Content.(type) {
		case []interface{}:
			for _, v := range content {
				if textMap, ok := v.(map[string]interface{}); ok && textMap["type"] == "text" {
					text, _ := textMap["text"].(string)
					contentBuilder.WriteString(text)
				}
			}
		case string:
			contentBuilder.WriteString(message.Role + ":" + wrapUrlsToTags(content) + "\n")
		}
	}

	return contentBuilder.String()
}
