package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/cphovo/ollm/sydney"
	"github.com/cphovo/ollm/util"
	"github.com/gin-gonic/gin"
)

type ImageUploadRequest struct {
	Cookies string                `form:"cookies"`
	File    *multipart.FileHeader `form:"file"`
}

func BingImageUploadHandler(c *gin.Context) {
	var uploadReq ImageUploadRequest
	if err := c.ShouldBind(&uploadReq); err != nil {
		c.String(http.StatusBadRequest, "Bad request: %v", err)
		return
	}

	cookies := util.Ternary(uploadReq.Cookies == "", DefaultCookies, util.ParseCookies(uploadReq.Cookies))

	file, err := uploadReq.File.Open()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to open the file: %v", err)
		return
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read the file: %v", err)
		return
	}

	// Upload image
	imgUrl, err := sydney.NewSydney(sydney.Options{
		Cookies: cookies,
		Proxy:   Proxy,
	}).UploadImage(bytes)

	if err != nil {
		c.String(http.StatusInternalServerError, "Image upload failed: %v", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"imgUrl": imgUrl,
	})
}

func BingImageCreateHandler(c *gin.Context) {
	var request sydney.CreateImageRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cookies := util.Ternary(request.Cookies == "", DefaultCookies, util.ParseCookies(request.Cookies))

	// Create image
	image, err := sydney.NewSydney(sydney.Options{
		Cookies:           cookies,
		Proxy:             Proxy,
		ConversationStyle: "Creative",
	}).GenerateImage(request.Image)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, image)
}

func BingStreamChatHandler(c *gin.Context) {
	var request sydney.ChatStreamRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cookies := util.Ternary(request.Cookies == "", DefaultCookies, util.ParseCookies(request.Cookies))

	sydneyAPI := sydney.NewSydney(sydney.Options{
		Cookies:           cookies,
		Proxy:             Proxy,
		ConversationStyle: request.ConversationStyle,
		NoSearch:          request.NoSearch,
		GPT4Turbo:         request.UseGPT4Turbo,
		UseClassic:        request.UseClassic,
		Plugins:           request.Plugins,
	})

	// Stream chat
	messageCh, err := sydneyAPI.AskStream(sydney.AskStreamOptions{
		StopCtx:        c.Request.Context(),
		Prompt:         request.Prompt,
		WebpageContext: request.WebpageContext,
		ImageURL:       request.ImageURL,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating conversation: " + err.Error()})
		return
	}

	c.Stream(func(w io.Writer) bool {
		for message := range messageCh {
			encoded, _ := json.Marshal(message.Text)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", message.Type, encoded)
			c.Writer.Flush()
		}
		return false
	})
}

func BingCompleteChatHandler(c *gin.Context) {
	var request sydney.OpenAIChatCompletionRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parsedMessages, err := sydney.ParseOpenAIMessages(request.Messages)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cookiesStr := c.GetHeader("Cookie")
	cookies := util.Ternary(cookiesStr == "", DefaultCookies, util.ParseCookies(cookiesStr))

	conversationStyle := util.Ternary(
		strings.HasPrefix(request.Model, "gpt-3.5-turbo"), "Balanced", request.Model)

	sydneyAPI := sydney.NewSydney(sydney.Options{
		Cookies:           cookies,
		Proxy:             Proxy,
		ConversationStyle: conversationStyle,
		Locale:            "en-US",
		NoSearch:          request.ToolChoice == nil,
		GPT4Turbo:         true,
	})

	messageCh, err := sydneyAPI.AskStream(sydney.AskStreamOptions{
		StopCtx:        c.Request.Context(),
		Prompt:         parsedMessages.Prompt,
		WebpageContext: parsedMessages.WebpageContext,
		ImageURL:       parsedMessages.ImageURL,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating conversation: " + err.Error()})
		return
	}

	if !request.Stream {
		var replyBuilder strings.Builder
		errored := false

		for message := range messageCh {
			switch message.Type {
			case sydney.MessageTypeMessageText:
				replyBuilder.WriteString(message.Text)
			case sydney.MessageTypeError:
				errored = true
				replyBuilder.WriteString("`Error: ")
				replyBuilder.WriteString(message.Text)
				replyBuilder.WriteString("`")
			}
		}

		c.JSON(http.StatusOK, sydney.NewOpenAIChatCompletion(
			conversationStyle,
			replyBuilder.String(),
			util.Ternary(errored, sydney.FinishReasonLength, sydney.FinishReasonStop),
		))

		return
	}

	c.Stream(func(w io.Writer) bool {
		errored := false

		for message := range messageCh {
			var delta string

			switch message.Type {
			case sydney.MessageTypeMessageText:
				delta = message.Text
			case sydney.MessageTypeError:
				errored = true
				delta = fmt.Sprintf("`Error: %s`", message.Text)
			default:
				continue
			}

			chunk := sydney.NewOpenAIChatCompletionChunk(conversationStyle, delta, nil)
			encoded, err := json.Marshal(chunk)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", encoded)
			c.Writer.Flush()
		}

		chunk := sydney.NewOpenAIChatCompletionChunk(conversationStyle, "", util.Ternary(errored, &sydney.FinishReasonLength, &sydney.FinishReasonStop))
		encoded, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n", encoded)
		c.Writer.Flush()

		return false
	})
}

func BingGenerateImageHandler(c *gin.Context) {
	var request sydney.OpenAIImageGenerationRequest

	// Bind JSON request body to struct
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cookiesStr := c.GetHeader("Cookie")
	cookies := util.Ternary(cookiesStr == "", DefaultCookies, util.ParseCookies(cookiesStr))

	sydneyAPI := sydney.NewSydney(sydney.Options{
		Cookies:           cookies,
		Proxy:             Proxy,
		ConversationStyle: "Creative",
		Locale:            "en-US",
	})

	// Ask stream with a new context
	newContext, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	messageCh, err := sydneyAPI.AskStream(sydney.AskStreamOptions{
		StopCtx:        newContext,
		Prompt:         "Create image for the description: " + request.Prompt,
		WebpageContext: sydney.ImageGeneratorContext, // Assuming ImageGeneratorContext is predefined
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var generativeImage sydney.GenerativeImage

	for message := range messageCh {
		if message.Type == sydney.MessageTypeGenerativeImage {
			err := json.Unmarshal([]byte(message.Text), &generativeImage)
			if err == nil {
				break
			}
		}
	}
	cancel()

	if generativeImage.URL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "empty generative image"})
		return
	}

	image, err := sydneyAPI.GenerateImage(generativeImage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sydney.ToOpenAIImageGeneration(image))
}
