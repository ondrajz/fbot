package main

import (
	"fmt"
	"os"

	"github.com/gookit/color"
	"github.com/sirupsen/logrus"
)

const (
	EnvVarLogLevel = "FBOT_LOGLVL"
	EnvVarDebug    = "FBOT_DEBUG"

	EnvVarOpenAiApiKey        = "FBOT_OPENAI_API_KEY"
	EnvVarTelegramBotApiToken = "FBOT_TELEGRAM_BOT_API_TOKEN"
)

type Config struct {
	Debug    string
	LogLevel string

	OpenAI struct {
		ApiKey string
	}
	Telegram struct {
		BotApiToken string
	}
}

func LoadConfigEnv() (*Config, error) {
	cfg := &Config{
		Debug:    os.Getenv(EnvVarDebug),
		LogLevel: os.Getenv(EnvVarLogLevel),
	}

	cfg.OpenAI.ApiKey = os.Getenv(EnvVarOpenAiApiKey)
	cfg.Telegram.BotApiToken = os.Getenv(EnvVarTelegramBotApiToken)

	return cfg, nil
}

func validateConfig(cfg *Config) error {
	logrus.Tracef("Config: %s", color.Gray.Sprint(toJson(cfg)))
	if cfg.OpenAI.ApiKey == "" {
		return fmt.Errorf("openAI API key is required")
	}
	if cfg.Telegram.BotApiToken == "" {
		return fmt.Errorf("telegram bot API token is required")
	}
	return nil
}
