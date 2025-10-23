package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type Config struct {
	TelegramBotToken string `name:"telegram"`
}

var cfg Config

func init() {
	cfg = Config{
		TelegramBotToken: "",
	}
}

func (c *Config) Init() {

	// Initialize API keys
	loadAPIKeys()
}

func GetConfig() *Config {
	return &cfg
}

// loadAPIKey loads the OpenAI API key from config file
func loadAPIKeys() (string, error) {
	// Try relative path first
	keyPath := "config/api_keys.txt"
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		// Try absolute path
		cwd, _ := os.Getwd()
		keyPath = filepath.Join(cwd, "config", "api_keys.txt")
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("API key file not found at %s. Please create the file with your API keys. See config/api_keys.example for format: %v", keyPath, err)
	}

	apiKey := strings.TrimSpace(string(data))
	lines := strings.Split(string(data), "\n")

	keys := &cfg
	val := reflect.ValueOf(keys).Elem()
	typ := val.Type()

	//TODO: better parsing and error handling
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		fieldName := strings.TrimSpace(parts[0])
		fieldValue := strings.TrimSpace(parts[1])

		for i := 0; i < typ.NumField(); i++ {
			tag := typ.Field(i).Tag.Get("name")
			if tag == fieldName {
				val.Field(i).SetString(fieldValue)
			}
		}
	}

	return apiKey, nil
}
