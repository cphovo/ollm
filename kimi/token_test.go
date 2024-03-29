package kimi

import (
	"fmt"
	"testing"
)

func TestGetToken(t *testing.T) {
	var refreshToken = "eyJhbGciOiJIUzUxMiI..."
	resp, err := GetToken(refreshToken)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resp)
}
