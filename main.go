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
	// load envs
	err := util.LoadEnv(".env")
	if err != nil {
		fmt.Println("Failed to load .env file:", err)
		return
	}

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

	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		panic("在这里提供你默认的 Gemini API Key")
	}

	handler.Proxy = proxy
	handler.DefaultCookies = defaultCookies
	handler.DefaultRefreshToken = refreshToken
	handler.DefaultAPIKey = geminiAPIKey

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

	// GEMINI AI
	r.POST("/gemini/chat/stream", handler.GeminiStreamChatHandler)
	r.POST("/v1/gemini/chat/completions", handler.GeminiCompleteChatHandler)

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
