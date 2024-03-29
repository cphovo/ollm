package kimi

import "net/http"

var CommonHeaders = map[string]string{
	"accept":             "*/*",
	"accept-language":    "zh-CN,zh;q=0.9",
	"origin":             "https://kimi.moonshot.cn",
	"content-type":       "application/json",
	"r-timezone":         "Asia/Shanghai",
	"sec-ch-ua":          "\"Chromium\";v=\"122\", \"Not(A:Brand\";v=\"24\", \"Google Chrome\";v=\"122\"",
	"sec-ch-ua-mobile":   "?0",
	"sec-ch-ua-platform": "\"macOS\"",
	"sec-fetch-dest":     "empty",
	"sec-fetch-mode":     "cors",
	"sec-fetch-site":     "same-origin",
	"user-agent":         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
}

func SetCommonHeaders(req *http.Request) {
	for k, v := range CommonHeaders {
		req.Header.Set(k, v)
	}
}
