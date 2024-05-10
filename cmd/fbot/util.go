package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func toYaml(v any) string {
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return err.Error()
	}
	return buf.String()
}

func toJson(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func downloadImageFromURL(url string) (string, error) {
	// Create a temporary file to save the image to
	file, err := os.CreateTemp("", "image-*.jpg")
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Download the image from the URL
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Save the image to the temporary file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}

	// Return the path to the temporary file
	return file.Name(), nil
}
