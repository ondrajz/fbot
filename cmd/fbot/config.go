package main

import "os"

var (
	openAiApiKey   = os.Getenv("FBOT_OPENAI_API_KEY")
	telegramApiKey = os.Getenv("FBOT_TELEGRAM_BOT_API_TOKEN")
)

type Config struct {
	OpenAI struct {
		ApiKey string
	}
	Telegram struct {
		BotApiToken string
	}
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	cfg.OpenAI.ApiKey = openAiApiKey
	cfg.Telegram.BotApiToken = telegramApiKey

	return cfg, nil
}
