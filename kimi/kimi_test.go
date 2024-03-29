package kimi

import (
	"fmt"
	"testing"
)

var refreshToken = "eyJhbGciOiJIUzUxMiI..."

func TestKimiAskStream(t *testing.T) {
	kimi, err := NewKimi(refreshToken)
	if err != nil {
		fmt.Println(err)
	}

	convId, err := kimi.CreateChat("未命名会话")
	if err != nil {
		fmt.Println(err)
	}

	messages, err := kimi.AskStream(AskStreamOptions{
		Text:      "hello",
		ConvId:    convId,
		UseSearch: true,
	})

	if err != nil {
		fmt.Println(err)
	}

	for message := range messages {
		data := fmt.Sprintf("event: %s\ndata: %s\n", message.Event, message.Text)
		fmt.Println(data)
	}
}
