package main

import (
	"github.com/otiai10/gosseract/v2"
	"github.com/sirupsen/logrus"
)

func detectTextFromImage(file string) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()

	langs, _ := gosseract.GetAvailableLanguages()
	logrus.Debug(langs)
	if err := client.SetLanguage("eng", "slk"); err != nil {
		return "", err
	}

	ver := client.Version()
	logrus.Debugf("serract server version: %v", ver)

	if err := client.SetImage(file); err != nil {
		return "", err
	}
	text, err := client.Text()
	if err != nil {
		return "", err
	}

	return text, nil
}
