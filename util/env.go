package util

import (
	"bufio"
	"os"
	"strings"
)

func LoadEnv(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") { // 忽略空行和注释
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // 忽略不合法的行
		}

		key := parts[0]
		value := parts[1]

		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return scanner.Err()
}
