package main

import (
	"testing"
)

func TestImage(t *testing.T) {
	text, err := detectText("/home/ondrej/Pictures/chatgpt-whatsapp-school-export.png")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("detected text: %v", text)

}
