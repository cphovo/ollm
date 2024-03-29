package kimi

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

const (
	userUrl       = "https://kimi.moonshot.cn/api/user"
	userStatusUrl = "https://kimi.moonshot.cn/api/chat_1m/user/status"
	chatListUrl   = "https://kimi.moonshot.cn/api/chat/list"
)

// 模拟用户的一些请求，正常使用 Kimi 官网时，时不时的会触发以下 3 种请求
func FakeUserRequest(accessToken string) error {
	client := &http.Client{}

	requests := []func() (*http.Response, error){
		func() (*http.Response, error) {
			req, _ := http.NewRequest("GET", userUrl, nil)
			SetCommonHeaders(req)
			req.Header.Add("Authorization", "Bearer "+accessToken)
			req.Header.Add("Referer", "https://kimi.moonshot.cn/")
			return client.Do(req)
		},
		func() (*http.Response, error) {
			req, _ := http.NewRequest("GET", userStatusUrl, nil)
			SetCommonHeaders(req)
			req.Header.Add("Authorization", "Bearer "+accessToken)
			req.Header.Add("Referer", "https://kimi.moonshot.cn/")
			return client.Do(req)
		},
		func() (*http.Response, error) {
			req, _ := http.NewRequest("POST", chatListUrl, bytes.NewBuffer([]byte(`{"offset":0, "size":50}`)))
			SetCommonHeaders(req)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Add("Authorization", "Bearer "+accessToken)
			req.Header.Add("Referer", "https://kimi.moonshot.cn/")
			return client.Do(req)
		},
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	index := r.Intn(len(requests))
	resp, err := requests[index]()
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to fake request with status code: %d", resp.StatusCode)
	}

	return nil
}
