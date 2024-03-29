package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/cphovo/ollm/kimi"
	"github.com/cphovo/ollm/sydney"
	"github.com/cphovo/ollm/util"
	"github.com/gin-gonic/gin"
)

type KimiChatStreamRequest struct {
	Text         string `json:"text"`
	ConvId       string `json:"convId"`
	RefreshToken string `json:"refreshToken"`
	UseSearch    bool   `json:"useSearch"`
}

type KimiOpenAIMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type KimiOpenAIChatCompletionRequest struct {
	Model        string              `json:"model"`
	Messages     []KimiOpenAIMessage `json:"messages"`
	Stream       bool                `json:"stream"`
	ToolChoice   *interface{}        `json:"tool_choice"`
	RefreshToken string              `json:"refreshToken"`
	UseSearch    *bool               `json:"useSearch"`
}

func KimiStreamChatHandler(c *gin.Context) {
	var request KimiChatStreamRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Text is required"})
		return
	}

	refreshToken := util.Ternary(request.RefreshToken == "", DefaultRefreshToken, request.RefreshToken)

	kimiAI, err := kimi.NewKimi(refreshToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// new conversation
	if request.ConvId == "" {
		convId, err := kimiAI.CreateChat("未命名会话")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating conversation: " + err.Error()})
			return
		}
		request.ConvId = convId
	}

	messageCh, err := kimiAI.AskStream(kimi.AskStreamOptions{
		Text:      request.Text,
		ConvId:    request.ConvId,
		UseSearch: request.UseSearch,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating conversation: " + err.Error()})
		return
	}

	c.Stream(func(w io.Writer) bool {
		for message := range messageCh {
			encoded, _ := json.Marshal(message.Text)
			// 将 cmpl 类型转换成 message 类型消息，方便和 bing 统一
			if message.Event == "cmpl" {
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", "message", encoded)
			} else {
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", message.Event, encoded)
			}
			c.Writer.Flush()
		}

		// 最后追加一个 conv_id 事件
		encoded, _ := json.Marshal(request.ConvId)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", "conv_id", encoded)
		c.Writer.Flush()

		return false
	})
}

func KimiCompleteChatHandler(c *gin.Context) {
	var request KimiOpenAIChatCompletionRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(request.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Messages is required"})
		return
	}

	refreshToken := util.Ternary(request.RefreshToken == "", DefaultRefreshToken, request.RefreshToken)
	useSearch := util.Ternary(request.UseSearch != nil, *request.UseSearch, true)

	kimiAI, err := kimi.NewKimi(refreshToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 没次创建一个新聊天
	convId, err := kimiAI.CreateChat("未命名会话")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating conversation: " + err.Error()})
		return
	}

	// 将 OpenAI 格式的消息转换成 Kimi 格式的消息
	text := messagesPrepare(request.Messages)

	messageCh, err := kimiAI.AskStream(kimi.AskStreamOptions{
		Text:      text,
		ConvId:    convId,
		UseSearch: useSearch,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating conversation: " + err.Error()})
		return
	}

	if !request.Stream {
		var replyBuilder strings.Builder
		errored := false

		for message := range messageCh {
			// TODO
			switch message.Event {
			case "cmpl":
				replyBuilder.WriteString(message.Text)
			case "error":
				errored = true
				replyBuilder.WriteString("`Error: ")
				replyBuilder.WriteString(message.Text)
				replyBuilder.WriteString("`")
			}
		}
		c.JSON(http.StatusOK, sydney.NewOpenAIChatCompletion(
			"KIMI",
			replyBuilder.String(),
			util.Ternary(errored, sydney.FinishReasonLength, sydney.FinishReasonStop),
		))

		return
	}

	c.Stream(func(w io.Writer) bool {
		errored := false

		for message := range messageCh {
			var delta string

			switch message.Event {
			case "cmpl":
				delta = message.Text
			case "error":
				errored = true
				delta = fmt.Sprintf("`Error: %s`", message.Text)
			default:
				continue
			}

			chunk := sydney.NewOpenAIChatCompletionChunk("KIMI", delta, nil)
			encoded, err := json.Marshal(chunk)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", encoded)
			c.Writer.Flush()
		}

		chunk := sydney.NewOpenAIChatCompletionChunk("KIMI", "", util.Ternary(errored, &sydney.FinishReasonLength, &sydney.FinishReasonStop))
		encoded, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n", encoded)
		c.Writer.Flush()

		return false
	})
}

func wrapUrlsToTags(content string) string {
	re := regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)`)
	return re.ReplaceAllString(content, `<url id="" type="url" status="" title="" wc="">$0</url>`)
}

func messagesPrepare(messages []KimiOpenAIMessage) string {
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
