package main

import (
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

const (
	defaultModel     = openai.GPT3Dot5Turbo
	defaultMaxTokens = 1024
)

func (fbot *FBot) initOpenAI(apiKey string) error {
	// Create a new OpenAI client using the provided API key
	client := openai.NewClient(apiKey)

	fbot.openaiClient = client

	if err := fbot.loadOpenAiModels(); err != nil {
		return fmt.Errorf("loading OpenAI models failed: %w", err)
	}
	logrus.Debugf("OpenAI client activated")

	return nil
}

func (fbot *FBot) loadOpenAiModels() error {
	listResp, err := fbot.openaiClient.ListModels(fbot.ctx)
	if err != nil {
		return fmt.Errorf("ListModels error: %v", err)
	}
	logrus.Debugf("loaded %d OpenAI models", len(listResp.Models))

	fbot.aiModels = listResp.Models

	if fbot.isDebugOnFor("openai") {
		for i, model := range listResp.Models {
			idx := i + 1
			logrus.Tracef(" - %3d - %v %v (ownedBy: %v) %+v", idx, model.Object, model.ID, model.OwnedBy, model.Permission)
		}
	}
	return nil
}

func (fbot *FBot) sendAIChatRequest(chatMessages []openai.ChatCompletionMessage) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:     fbot.selectedModel,
		Messages:  chatMessages,
		MaxTokens: fbot.modelMaxTokens,
	}

	logrus.Tracef("sending chat completion request:\n%s", toJson(req))

	t0 := time.Now()

	resp, err := fbot.openaiClient.CreateChatCompletion(fbot.ctx, req)
	if err != nil {
		return "", fmt.Errorf("CreateChatCompletion error: %w", err)
	}

	took := time.Since(t0)

	logrus.Tracef("chat completion done (took %v) response: %v\n\n", took, toJson(resp))

	respMsg := resp.Choices[0].Message.Content

	if len(resp.Choices) > 1 {
		logrus.Debugf("chat completion response (%d choices):\n", len(resp.Choices))
		for _, choice := range resp.Choices {
			logrus.Debugf(" - choice %2d: %s", choice.Index, choice.Message)
		}
	} else {
		logrus.Debugf("chat completion response:\n%s", respMsg)
	}

	return respMsg, nil
}

func newUserCompletionMessage(content string) []openai.ChatCompletionMessage {
	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: content},
	}
}
