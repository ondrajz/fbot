package main

import (
	"fmt"

	"github.com/gookit/color"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := runBot(); err != nil {
		logrus.Fatalf("ERROR: %v", err)
	}
}

func runBot() error {
	// Load configuration
	cfg, err := LoadConfigEnv()
	if err != nil {
		return fmt.Errorf("loading config failed: %w", err)
	}

	SetLogLevel(cfg.LogLevel)

	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Setup bot
	fbot, err := NewFBot(cfg)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	logrus.Info(color.Green.Sprint("FBot is online!"))

	fbot.processUpdates()

	return nil
}
