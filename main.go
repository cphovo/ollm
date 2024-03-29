package main

import (
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/cphovo/ollm/handler"
	"github.com/cphovo/ollm/util"
	"github.com/gin-gonic/gin"
)

type ImageUploadRequest struct {
	Cookies string                `form:"cookies"`
	File    *multipart.FileHeader `form:"file"`
}

var (
	port string
	// proxy          string
	allowedOrigins string
	// defaultCookies map[string]string
	authToken string
)

func init() {
	// read envs
	port = os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	proxy := os.Getenv("HTTPS_PROXY")
	if proxy == "" {
		proxy = os.Getenv("HTTP_PROXY")
	}

	allowedOrigins = os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}

	defaultCookies := util.ParseCookies(os.Getenv("DEFAULT_COOKIES"))

	if len(defaultCookies) == 0 {
		slog.Info("DEFAULT_COOKIES not set, reading from cookies.json")
		defaultCookies, _ = util.ReadCookiesFile()
		if len(defaultCookies) == 0 {
			slog.Warn("cookies.json not found, using empty cookies")
		}
	} else {
		slog.Info("DEFAULT_COOKIES set, cookies.json will be ignored")
	}

	refreshToken := os.Getenv("KIMI_REFRESH_TOKEN")
	if refreshToken == "" {
		// refreshToken = "eyJhbGciOiJIUzUxM"
		panic("在这里提供你默认的 refreshToken")
	}

	handler.Proxy = proxy
	handler.DefaultCookies = defaultCookies
	handler.DefaultRefreshToken = refreshToken

	authToken = os.Getenv("AUTH_TOKEN")
}

func main() {
	r := gin.Default()

	r.Use(CORSMiddleware())
	r.Use(AuthMiddleware(authToken))

	r.GET("/", RootHandler)

	// BING AI
	r.POST("/image/upload", handler.BingImageUploadHandler)
	r.POST("/image/create", handler.BingImageCreateHandler)
	r.POST("/chat/stream", handler.BingStreamChatHandler)
	// TODO 合并路由，通过模型区分
	r.POST("/v1/chat/completions", handler.BingCompleteChatHandler)
	r.POST("/v1/images/generations", handler.BingGenerateImageHandler)

	// KIMI AI
	r.POST("/kimi/chat/stream", handler.KimiStreamChatHandler)
	r.POST("/v1/kimi/chat/completions", handler.KimiCompleteChatHandler)

	r.Run(fmt.Sprintf(":%s", port))
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", allowedOrigins)
		c.Header("Access-Control-Allow-Methods", "*")
		c.Header("Access-Control-Allow-Headers", "*")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func AuthMiddleware(authToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authToken == "" {
			c.Next()
			return
		}

		if c.GetHeader("Authorization") != "Bearer "+authToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		c.Next()
	}
}

func RootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Everything is OK!",
	})
}

// func imageUploadHandler(c *gin.Context) {
// 	var uploadReq ImageUploadRequest
// 	if err := c.ShouldBind(&uploadReq); err != nil {
// 		c.String(http.StatusBadRequest, "Bad request: %v", err)
// 		return
// 	}

// 	cookies := util.Ternary(uploadReq.Cookies == "", defaultCookies, util.ParseCookies(uploadReq.Cookies))

// 	file, err := uploadReq.File.Open()
// 	if err != nil {
// 		c.String(http.StatusInternalServerError, "Failed to open the file: %v", err)
// 		return
// 	}
// 	defer file.Close()

// 	bytes, err := io.ReadAll(file)
// 	if err != nil {
// 		c.String(http.StatusInternalServerError, "Failed to read the file: %v", err)
// 		return
// 	}

// 	// Upload image
// 	imgUrl, err := sydney.NewSydney(sydney.Options{
// 		Cookies: cookies,
// 		Proxy:   proxy,
// 	}).UploadImage(bytes)

// 	if err != nil {
// 		c.String(http.StatusInternalServerError, "Image upload failed: %v", err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"imgUrl": imgUrl,
// 	})
// }

// func imageCreateHandler(c *gin.Context) {
// 	var request sydney.CreateImageRequest

// 	if err := c.BindJSON(&request); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	cookies := util.Ternary(request.Cookies == "", defaultCookies, util.ParseCookies(request.Cookies))

// 	// Create image
// 	image, err := sydney.NewSydney(sydney.Options{
// 		Cookies:           cookies,
// 		Proxy:             proxy,
// 		ConversationStyle: "Creative",
// 	}).GenerateImage(request.Image)

// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, image)
// }

// func streamChatHandler(c *gin.Context) {
// 	var request sydney.ChatStreamRequest

// 	if err := c.BindJSON(&request); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	cookies := util.Ternary(request.Cookies == "", defaultCookies, util.ParseCookies(request.Cookies))

// 	sydneyAPI := sydney.NewSydney(sydney.Options{
// 		Cookies:           cookies,
// 		Proxy:             proxy,
// 		ConversationStyle: request.ConversationStyle,
// 		NoSearch:          request.NoSearch,
// 		GPT4Turbo:         request.UseGPT4Turbo,
// 		UseClassic:        request.UseClassic,
// 		Plugins:           request.Plugins,
// 	})

// 	// Stream chat
// 	messageCh, err := sydneyAPI.AskStream(sydney.AskStreamOptions{
// 		StopCtx:        c.Request.Context(),
// 		Prompt:         request.Prompt,
// 		WebpageContext: request.WebpageContext,
// 		ImageURL:       request.ImageURL,
// 	})
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "error creating conversation: " + err.Error()})
// 		return
// 	}

// 	c.Stream(func(w io.Writer) bool {
// 		for message := range messageCh {
// 			encoded, _ := json.Marshal(message.Text)
// 			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", message.Type, encoded)
// 			c.Writer.Flush()
// 		}
// 		return false
// 	})
// }

// func completeChatHandler(c *gin.Context) {
// 	var request sydney.OpenAIChatCompletionRequest

// 	if err := c.BindJSON(&request); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	parsedMessages, err := sydney.ParseOpenAIMessages(request.Messages)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	cookiesStr := c.GetHeader("Cookie")
// 	cookies := util.Ternary(cookiesStr == "", defaultCookies, util.ParseCookies(cookiesStr))

// 	conversationStyle := util.Ternary(
// 		strings.HasPrefix(request.Model, "gpt-3.5-turbo"), "Balanced", "Creative")

// 	sydneyAPI := sydney.NewSydney(sydney.Options{
// 		Cookies:           cookies,
// 		Proxy:             proxy,
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

// 		c.JSON(http.StatusOK, sydney.NewOpenAIChatCompletion(
// 			conversationStyle,
// 			replyBuilder.String(),
// 			util.Ternary(errored, sydney.FinishReasonLength, sydney.FinishReasonStop),
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

// 			chunk := sydney.NewOpenAIChatCompletionChunk(conversationStyle, delta, nil)
// 			encoded, err := json.Marshal(chunk)
// 			if err != nil {
// 				continue
// 			}

// 			fmt.Fprintf(w, "data: %s\n\n", encoded)
// 			c.Writer.Flush()
// 		}

// 		chunk := sydney.NewOpenAIChatCompletionChunk(conversationStyle, "", util.Ternary(errored, &sydney.FinishReasonLength, &sydney.FinishReasonStop))
// 		encoded, _ := json.Marshal(chunk)
// 		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n", encoded)
// 		c.Writer.Flush()

// 		return false
// 	})
// }

// func generateImageHandler(c *gin.Context) {
// 	var request sydney.OpenAIImageGenerationRequest

// 	// Bind JSON request body to struct
// 	if err := c.BindJSON(&request); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	cookiesStr := c.GetHeader("Cookie")
// 	cookies := util.Ternary(cookiesStr == "", defaultCookies, util.ParseCookies(cookiesStr))

// 	sydneyAPI := sydney.NewSydney(sydney.Options{
// 		Cookies:           cookies,
// 		Proxy:             proxy,
// 		ConversationStyle: "Creative",
// 		Locale:            "en-US",
// 	})

// 	// Ask stream with a new context
// 	newContext, cancel := context.WithCancel(c.Request.Context())
// 	defer cancel()

// 	messageCh, err := sydneyAPI.AskStream(sydney.AskStreamOptions{
// 		StopCtx:        newContext,
// 		Prompt:         "Create image for the description: " + request.Prompt,
// 		WebpageContext: sydney.ImageGeneratorContext, // Assuming ImageGeneratorContext is predefined
// 	})
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	var generativeImage sydney.GenerativeImage

// 	for message := range messageCh {
// 		if message.Type == sydney.MessageTypeGenerativeImage {
// 			err := json.Unmarshal([]byte(message.Text), &generativeImage)
// 			if err == nil {
// 				break
// 			}
// 		}
// 	}
// 	cancel()

// 	if generativeImage.URL == "" {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "empty generative image"})
// 		return
// 	}

// 	image, err := sydneyAPI.GenerateImage(generativeImage)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, sydney.ToOpenAIImageGeneration(image))
// }
