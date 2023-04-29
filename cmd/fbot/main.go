package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func main() {
	if err := runBot(); err != nil {
		logrus.Fatalln("ERROR: %v", err)
	}
}

func runBot() error {
	// Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config failed: %w", err)
	}

	// Setup bot
	fbot, err := NewFBot(cfg)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	fbot.processUpdates()

	return nil
}
