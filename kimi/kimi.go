package kimi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cphovo/ollm/util"
)

type Kimi struct {
	AccessToken  string
	RefreshToken string
	Store        *util.LazyExpiryKVStore
}

type AskStreamOptions struct {
	Text   string
	ConvId string
	// AccessToken string
	UseSearch bool
}

type Message struct {
	Event string
	Text  string
}

type KimiCreateChatPayload struct {
	Name      string `json:"name"`
	IsExample bool   `json:"is_example"`
}

type KimiCreateChatResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	IsExample bool   `json:"is_example"`
	Status    string `json:"status"`
	Type      string `json:"type"`
}

const (
	KimiCompletionStreamURL = "https://kimi.moonshot.cn/api/chat/%s/completion/stream"
	KimiCreateChatURL       = "https://kimi.moonshot.cn/api/chat"
)

var globalStore = util.NewLazyExpiryStore()

func NewKimi(refreshToken string) (*Kimi, error) {
	// 如果缓存中存在未失效的 TOKEN，直接使用
	if token, ok := globalStore.Get(refreshToken); ok {
		tokenResp := token.(KimiTokenResponse)
		return &Kimi{
			AccessToken:  tokenResp.AccessToken,
			RefreshToken: tokenResp.RefreshToken,
			Store:        globalStore,
		}, nil
	}

	// 否则获取新的 TOKEN
	tokenResponse, err := GetToken(refreshToken)
	if err != nil {
		return nil, err
	}

	kimi := &Kimi{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
		Store:        globalStore,
	}

	kimi.Store.Set(refreshToken, tokenResponse, KimiTokenExpireTime)

	return kimi, nil
}

func (kimi *Kimi) CreateChat(name string) (string, error) {
	payload := KimiCreateChatPayload{
		Name:      name,
		IsExample: false,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling payload: %w", err)
	}

	req, err := http.NewRequest("POST", KimiCreateChatURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+kimi.AccessToken)
	req.Header.Set("Referer", "https://kimi.moonshot.cn/")
	SetCommonHeaders(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to create kimi chat, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response KimiCreateChatResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	return response.ID, nil
}

func (kimi *Kimi) AskStream(options AskStreamOptions) (<-chan Message, error) {
	url := fmt.Sprintf(KimiCompletionStreamURL, options.ConvId)

	payload := map[string]interface{}{
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": options.Text,
			},
		},
		"refs":       []string{},
		"use_search": options.UseSearch,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+kimi.AccessToken)
	req.Header.Set("Referer", "https://kimi.moonshot.cn/chat/"+options.ConvId)
	SetCommonHeaders(req)

	client := &http.Client{
		Timeout: 120 * time.Second, // 设置 120 超时时间
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	messages := make(chan Message)

	go func() {
		defer close(messages)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println("Error reading line:", err)
				return // 这将只结束 goroutine
			}

			if bytes.HasPrefix(line, []byte("data:")) {
				var data Message
				jsonStr := line[len("data:"):]
				if err := json.Unmarshal(jsonStr, &data); err != nil {
					fmt.Println("Error unmarshalling data:", err)
					continue
				}

				messages <- data
			}
		}
	}()

	return messages, nil
}
