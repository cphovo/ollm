package kimi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	KimiRefreshTokenURL = "https://kimi.moonshot.cn/api/auth/token/refresh"
	// Token 过期时间设置为 300 秒
	KimiTokenExpireTime = 5 * 60
)

type KimiTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func GetToken(kimiAuthToken string) (tokenResponse KimiTokenResponse, err error) {

	req, err := http.NewRequest("GET", KimiRefreshTokenURL, nil)
	if err != nil {
		return
	}

	SetCommonHeaders(req)
	req.Header.Add("Authorization", "Bearer "+kimiAuthToken)
	req.Header.Add("Referer", "https://kimi.moonshot.cn/")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return tokenResponse, fmt.Errorf("failed to get kimi refresh token, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// var tokenResponse KimiTokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return
	}

	return tokenResponse, nil
}
