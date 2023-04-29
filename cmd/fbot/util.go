package main

import (
	"encoding/json"
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
