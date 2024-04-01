package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	DefaultCookies      map[string]string
	Proxy               string
	DefaultRefreshToken string
	DefaultAPIKey       string
)

var HandlerMap = map[string]gin.HandlerFunc{
	"Creative":      BingCompleteChatHandler,
	"Balanced":      BingCompleteChatHandler,
	"Precise":       BingCompleteChatHandler,
	"gpt-3.5-turbo": BingCompleteChatHandler,
	"kimi":          KimiCompleteChatHandler,
	"gemini-pro":    GeminiCompleteChatHandler,
}

func ModelBasedDispatcher() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body OpenAIChatCompletionRequest
		data, _ := io.ReadAll(c.Request.Body)
		// 重置请求体，以便后续的 handler 可以再次读取
		c.Request.Body = io.NopCloser(bytes.NewReader(data))

		err := json.Unmarshal(data, &body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		if handler, exists := HandlerMap[body.Model]; exists {
			handler(c)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "model not supported"})
		}
	}
}

type OpenAIMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type CreateConversationResult struct {
	Value   string `json:"value"`
	Message string `json:"message"`
}

type CreateConversationResponse struct {
	ConversationId        string                   `json:"conversationId"`
	ClientId              string                   `json:"clientId"`
	Result                CreateConversationResult `json:"result"`
	SecAccessToken        string                   `json:"secAccessToken"`
	ConversationSignature string                   `json:"conversationSignature"`
	BearerToken           string                   `json:"bearerToken"`
}

type OpenAIChatCompletionRequest struct {
	Model        string                     `json:"model"`
	Messages     []OpenAIMessage            `json:"messages"`
	Stream       bool                       `json:"stream"`
	ToolChoice   *interface{}               `json:"tool_choice"`
	Conversation CreateConversationResponse `json:"conversation"`
	RefreshToken string                     `json:"refreshToken"` // Kimi
	UseSearch    *bool                      `json:"useSearch"`    // Kimi
	APIKey       string                     `json:"apiKey"`       // Gemini
}

// type ChoiceMessage struct {
// 	Content string `json:"content"`
// 	Role    string `json:"role"`
// }

// type ChatCompletionChoice struct {
// 	Index        int           `json:"index"`
// 	Message      ChoiceMessage `json:"message"`
// 	FinishReason string        `json:"finish_reason"`
// }

// type OpenAIChatCompletion struct {
// 	ID                string                 `json:"id"`
// 	Object            string                 `json:"object"`
// 	Created           int64                  `json:"created"`
// 	Model             string                 `json:"model"`
// 	SystemFingerprint string                 `json:"system_fingerprint"`
// 	Choices           []ChatCompletionChoice `json:"choices"`
// 	Usage             UsageStats             `json:"usage"`
// }

// type ChoiceDelta struct {
// 	Role    string `json:"role"`
// 	Content string `json:"content"`
// }

// type ChatCompletionChunkChoice struct {
// 	Index        int         `json:"index"`
// 	Delta        ChoiceDelta `json:"delta"`
// 	FinishReason *string     `json:"finish_reason"`
// }

// type OpenAIChatCompletionChunk struct {
// 	ID                string                      `json:"id"`
// 	Object            string                      `json:"object"`
// 	Created           int64                       `json:"created"`
// 	Model             string                      `json:"model"`
// 	SystemFingerprint string                      `json:"system_fingerprint"`
// 	Choices           []ChatCompletionChunkChoice `json:"choices"`
// }

// type UsageStats struct {
// 	PromptTokens     int `json:"prompt_tokens"`
// 	CompletionTokens int `json:"completion_tokens"`
// 	TotalTokens      int `json:"total_tokens"`
// }

// type OpenAIMessagesParseResult struct {
// 	WebpageContext string
// 	Prompt         string
// 	ImageURL       string
// }

// func chatWithBing(c *gin.Context, request OpenAIChatCompletionRequest) {
// 	parsedMessages, err := ParseOpenAIMessages(request.Messages)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	cookiesStr := c.GetHeader("Cookie")
// 	cookies := util.Ternary(cookiesStr == "", DefaultCookies, util.ParseCookies(cookiesStr))

// 	conversationStyle := util.Ternary(
// 		strings.HasPrefix(request.Model, "gpt-3.5-turbo"), "Balanced", "Creative")

// 	sydneyAPI := sydney.NewSydney(sydney.Options{
// 		Cookies:           cookies,
// 		Proxy:             Proxy,
// 		ConversationStyle: conversationStyle,
// 		Locale:            "en-US",
// 		NoSearch:          request.ToolChoice == nil,
// 		GPT4Turbo:         true,
// 	})

// 	messageCh, err := sydneyAPI.AskStream(sydney.AskStreamOptions{
// 		StopCtx:        c.Request.Context(),
// 		Prompt:         parsedMessages.Prompt,
// 		WebpageContext: parsedMessages.WebpageContext,
// 		ImageURL:       parsedMessages.ImageURL,
// 	})
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating conversation: " + err.Error()})
// 		return
// 	}

// 	if !request.Stream {
// 		var replyBuilder strings.Builder
// 		errored := false

// 		for message := range messageCh {
// 			switch message.Type {
// 			case sydney.MessageTypeMessageText:
// 				replyBuilder.WriteString(message.Text)
// 			case sydney.MessageTypeError:
// 				errored = true
// 				replyBuilder.WriteString("`Error: ")
// 				replyBuilder.WriteString(message.Text)
// 				replyBuilder.WriteString("`")
// 			}
// 		}

// 		c.JSON(http.StatusOK, NewOpenAIChatCompletion(
// 			conversationStyle,
// 			replyBuilder.String(),
// 			util.Ternary(errored, FinishReasonLength, FinishReasonStop),
// 		))

// 		return
// 	}

// 	c.Stream(func(w io.Writer) bool {
// 		errored := false

// 		for message := range messageCh {
// 			var delta string

// 			switch message.Type {
// 			case sydney.MessageTypeMessageText:
// 				delta = message.Text
// 			case sydney.MessageTypeError:
// 				errored = true
// 				delta = fmt.Sprintf("`Error: %s`", message.Text)
// 			default:
// 				continue
// 			}

// 			chunk := NewOpenAIChatCompletionChunk(conversationStyle, delta, nil)
// 			encoded, err := json.Marshal(chunk)
// 			if err != nil {
// 				continue
// 			}

// 			fmt.Fprintf(w, "data: %s\n\n", encoded)
// 			c.Writer.Flush()
// 		}

// 		chunk := NewOpenAIChatCompletionChunk(conversationStyle, "", util.Ternary(errored, &FinishReasonLength, &FinishReasonStop))
// 		encoded, _ := json.Marshal(chunk)
// 		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n", encoded)
// 		c.Writer.Flush()

// 		return false
// 	})
// }

// func chatWithKimi(c *gin.Context, request OpenAIChatCompletionRequest) {
// 	if len(request.Messages) == 0 {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Messages is required"})
// 		return
// 	}

// 	refreshToken := util.Ternary(request.RefreshToken == "", DefaultRefreshToken, request.RefreshToken)
// 	useSearch := util.Ternary(request.UseSearch != nil, *request.UseSearch, true)

// 	kimiAI, err := kimi.NewKimi(refreshToken)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	// 没次创建一个新聊天
// 	convId, err := kimiAI.CreateChat("未命名会话")
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating conversation: " + err.Error()})
// 		return
// 	}

// 	// 将 OpenAI 格式的消息转换成 Kimi 格式的消息
// 	text := ParseOpenAIMessagesToKimi(request.Messages)

// 	messageCh, err := kimiAI.AskStream(kimi.AskStreamOptions{
// 		Text:      text,
// 		ConvId:    convId,
// 		UseSearch: useSearch,
// 	})
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating conversation: " + err.Error()})
// 		return
// 	}

// 	if !request.Stream {
// 		var replyBuilder strings.Builder
// 		errored := false

// 		for message := range messageCh {
// 			// TODO
// 			switch message.Event {
// 			case "cmpl":
// 				replyBuilder.WriteString(message.Text)
// 			case "error":
// 				errored = true
// 				replyBuilder.WriteString("`Error: ")
// 				replyBuilder.WriteString(message.Text)
// 				replyBuilder.WriteString("`")
// 			}
// 		}
// 		c.JSON(http.StatusOK, sydney.NewOpenAIChatCompletion(
// 			"KIMI",
// 			replyBuilder.String(),
// 			util.Ternary(errored, sydney.FinishReasonLength, sydney.FinishReasonStop),
// 		))

// 		return
// 	}

// 	c.Stream(func(w io.Writer) bool {
// 		errored := false

// 		for message := range messageCh {
// 			var delta string

// 			switch message.Event {
// 			case "cmpl":
// 				delta = message.Text
// 			case "error":
// 				errored = true
// 				delta = fmt.Sprintf("`Error: %s`", message.Text)
// 			default:
// 				continue
// 			}

// 			chunk := sydney.NewOpenAIChatCompletionChunk("KIMI", delta, nil)
// 			encoded, err := json.Marshal(chunk)
// 			if err != nil {
// 				continue
// 			}

// 			fmt.Fprintf(w, "data: %s\n\n", encoded)
// 			c.Writer.Flush()
// 		}

// 		chunk := sydney.NewOpenAIChatCompletionChunk("KIMI", "", util.Ternary(errored, &sydney.FinishReasonLength, &sydney.FinishReasonStop))
// 		encoded, _ := json.Marshal(chunk)
// 		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n", encoded)
// 		c.Writer.Flush()

// 		return false
// 	})
// }

// func chatWithGemini(c *gin.Context, request OpenAIChatCompletionRequest) {
// 	if len(request.Messages) == 0 {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Messages is required"})
// 		return
// 	}

// 	model := util.Ternary(request.Model == "gemini", "gemini-pro", request.Model)
// 	apiKey := util.Ternary(request.APIKey == "", DefaultAPIKey, request.APIKey)

// 	// 将 OpenAI 格式的消息转换成 Kimi 格式的消息
// 	text := ParseOpenAIMessagesToKimi(request.Messages)

// 	messageCh, err := gemini.AskStream(gemini.AskStreamOptions{
// 		APIKey: apiKey,
// 		Model:  model,
// 		Prompt: text,
// 	})
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating conversation: " + err.Error()})
// 		return
// 	}

// 	if !request.Stream {
// 		var replyBuilder strings.Builder
// 		errored := false

// 		for message := range messageCh {
// 			// TODO
// 			switch message.Event {
// 			case "message":
// 				replyBuilder.WriteString(message.Text)
// 			case "error":
// 				errored = true
// 				replyBuilder.WriteString("`Error: ")
// 				replyBuilder.WriteString(message.Text)
// 				replyBuilder.WriteString("`")
// 			}
// 		}
// 		c.JSON(http.StatusOK, sydney.NewOpenAIChatCompletion(
// 			strings.ToUpper(model),
// 			replyBuilder.String(),
// 			util.Ternary(errored, sydney.FinishReasonLength, sydney.FinishReasonStop),
// 		))

// 		return
// 	}

// 	c.Stream(func(w io.Writer) bool {
// 		errored := false

// 		for message := range messageCh {
// 			var delta string

// 			switch message.Event {
// 			case "message":
// 				delta = message.Text
// 			case "error":
// 				errored = true
// 				delta = fmt.Sprintf("`Error: %s`", message.Text)
// 			default:
// 				continue
// 			}

// 			chunk := sydney.NewOpenAIChatCompletionChunk(strings.ToUpper(model), delta, nil)
// 			encoded, err := json.Marshal(chunk)
// 			if err != nil {
// 				continue
// 			}

// 			fmt.Fprintf(w, "data: %s\n\n", encoded)
// 			c.Writer.Flush()
// 		}

// 		chunk := sydney.NewOpenAIChatCompletionChunk(strings.ToUpper(model), "", util.Ternary(errored, &sydney.FinishReasonLength, &sydney.FinishReasonStop))
// 		encoded, _ := json.Marshal(chunk)
// 		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n", encoded)
// 		c.Writer.Flush()

// 		return false
// 	})
// }

// func ChatCompletionsHandler(c *gin.Context) {
// 	var request OpenAIChatCompletionRequest

// 	if err := c.BindJSON(&request); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	switch strings.ToLower(request.Model) {
// 	// Balanced Precise Creative
// 	case "creative", "balanced", "precise":
// 		chatWithBing(c, request)
// 	case "kimi":
// 		chatWithKimi(c, request)
// 	case "gemini":
// 		chatWithGemini(c, request)
// 	}
// }

// var (
// 	ErrMissingPrompt   = errors.New("user prompt is missing (last message is not sent by user)")
// 	FinishReasonStop   = "stop"
// 	FinishReasonLength = "length"
// )

// const (
// 	MessageRoleUser      = "user"
// 	MessageRoleAssistant = "assistant"
// 	MessageRoleSystem    = "system"
// )

// func ParseOpenAIMessages(messages []OpenAIMessage) (OpenAIMessagesParseResult, error) {
// 	if len(messages) == 0 {
// 		return OpenAIMessagesParseResult{}, ErrMissingPrompt
// 	}

// 	// find the last user message
// 	var promptIndex int
// 	var promptMessage OpenAIMessage

// 	for i := len(messages) - 1; i >= 0; i-- {
// 		if messages[i].Role == MessageRoleUser {
// 			promptIndex = i
// 			promptMessage = messages[i]
// 			break
// 		}
// 	}

// 	prompt, imageUrl := ParseOpenAIMessageContent(promptMessage.Content)

// 	if prompt == "" {
// 		return OpenAIMessagesParseResult{}, ErrMissingPrompt
// 	}

// 	if len(messages) == 1 {
// 		return OpenAIMessagesParseResult{
// 			Prompt:   prompt,
// 			ImageURL: imageUrl,
// 		}, nil
// 	}

// 	// exclude the promptMessage from the array
// 	messages = append(messages[:promptIndex], messages[promptIndex+1:]...)

// 	// construct context
// 	var contextBuilder strings.Builder
// 	contextBuilder.WriteString("\n\n")

// 	for i, message := range messages {
// 		// assert types
// 		text, _ := ParseOpenAIMessageContent(message.Content)

// 		// append role to context
// 		switch message.Role {
// 		case MessageRoleUser:
// 			contextBuilder.WriteString("[user](#message)\n")
// 		case MessageRoleAssistant:
// 			contextBuilder.WriteString("[assistant](#message)\n")
// 		case MessageRoleSystem:
// 			contextBuilder.WriteString("[system](#instructions)\n")
// 		default:
// 			continue // skip unknown roles
// 		}

// 		// append content to context
// 		contextBuilder.WriteString(text)
// 		if i != len(messages)-1 {
// 			contextBuilder.WriteString("\n\n")
// 		}
// 	}

// 	return OpenAIMessagesParseResult{
// 		Prompt:         prompt,
// 		WebpageContext: contextBuilder.String(),
// 		ImageURL:       imageUrl,
// 	}, nil
// }

// func ParseOpenAIMessageContent(content interface{}) (text, imageUrl string) {
// 	switch content := content.(type) {
// 	case string:
// 		// content is string, and it automatically becomes prompt
// 		text = content
// 	case []interface{}:
// 		// content is array of objects, and it contains prompt and optional image url
// 		for _, content := range content {
// 			content, ok := content.(map[string]interface{})
// 			if !ok {
// 				continue
// 			}
// 			switch content["type"] {
// 			case "text":
// 				if contentText, ok := content["text"].(string); ok {
// 					text = contentText
// 				}
// 			case "image_url":
// 				if url, ok := content["image_url"].(map[string]interface{}); ok {
// 					imageUrl, _ = url["url"].(string)
// 				}
// 			}
// 		}
// 	}

// 	return
// }

// func NewOpenAIChatCompletion(model, content, finishReason string) *OpenAIChatCompletion {
// 	return &OpenAIChatCompletion{
// 		ID:                "chatcmpl-123",
// 		Object:            "chat.completion",
// 		Created:           time.Now().Unix(),
// 		Model:             model,
// 		SystemFingerprint: "fp_44709d6fcb",
// 		Choices: []ChatCompletionChoice{
// 			{
// 				Index: 0,
// 				Message: ChoiceMessage{
// 					Role:    "assistant",
// 					Content: content,
// 				},
// 				FinishReason: finishReason,
// 			},
// 		},
// 		Usage: UsageStats{
// 			PromptTokens:     1024,
// 			CompletionTokens: 1024,
// 			TotalTokens:      2048,
// 		},
// 	}
// }

// func NewOpenAIChatCompletionChunk(model, delta string, finishReason *string) *OpenAIChatCompletionChunk {
// 	return &OpenAIChatCompletionChunk{
// 		ID:                "chatcmpl-123",
// 		Object:            "chat.completion",
// 		Created:           time.Now().Unix(),
// 		Model:             model,
// 		SystemFingerprint: "fp_44709d6fcb",
// 		Choices: []ChatCompletionChunkChoice{
// 			{
// 				Index: 0,
// 				Delta: ChoiceDelta{
// 					Role:    "assistant",
// 					Content: delta,
// 				},
// 				FinishReason: finishReason,
// 			},
// 		},
// 	}
// }

// func ParseUrlsToTags(content string) string {
// 	re := regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)`)
// 	return re.ReplaceAllString(content, `<url id="" type="url" status="" title="" wc="">$0</url>`)
// }

// func ParseOpenAIMessagesToKimi(messages []OpenAIMessage) string {
// 	var contentBuilder strings.Builder

// 	for _, message := range messages {
// 		switch content := message.Content.(type) {
// 		case []interface{}:
// 			for _, v := range content {
// 				if textMap, ok := v.(map[string]interface{}); ok && textMap["type"] == "text" {
// 					text, _ := textMap["text"].(string)
// 					contentBuilder.WriteString(text)
// 				}
// 			}
// 		case string:
// 			contentBuilder.WriteString(message.Role + ":" + ParseUrlsToTags(content) + "\n")
// 		}
// 	}

// 	return contentBuilder.String()
// }
