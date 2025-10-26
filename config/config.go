package config

import (
	"fmt"
	"time"
)

type Config struct {
	DeepSeek    AIConfig
	Qwen        AIConfig
	Gemini      AIConfig
	GeminiProxy string
	MaxRounds   int
	Timeout     time.Duration
}

type AIConfig struct {
	Model   string
	APIKey  string
	BaseURL string
}

func CheckTargetAIConfig(c *AIConfig) error {
	if c.APIKey == "" || c.BaseURL == "" || c.Model == "" {
		return fmt.Errorf("AI configuration is incomplete")
	}
	return nil
}

func CheckDriverAIConfig(c *AIConfig) error {
	if c.APIKey == "" || c.Model == "" {
		return fmt.Errorf("Driver AI configuration is incomplete")
	}
	return nil
}

func CheckConfig(c *Config) error {
	if err := CheckTargetAIConfig(&c.DeepSeek); err != nil {
		return fmt.Errorf("DeepSeek config error: %v", err)
	}
	if err := CheckTargetAIConfig(&c.Qwen); err != nil {
		return fmt.Errorf("Qwen config error: %v", err)
	}
	if err := CheckDriverAIConfig(&c.Gemini); err != nil {
		return fmt.Errorf("Gemini config error: %v", err)
	}
	if c.MaxRounds <= 0 {
		return fmt.Errorf("MaxRounds must be greater than 0")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("Timeout must be greater than 0")
	}
	return nil
}

func Load() (*Config, error) {
	Init()
	err := CheckConfig(config)
	if err != nil {
		return nil, err
	}
	config.Gemini.BaseURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/%s:generateContent?key=%s", config.Gemini.Model, config.Gemini.APIKey)

	return config, nil
}
