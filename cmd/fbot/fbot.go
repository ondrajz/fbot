package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/otiai10/gosseract/v2"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

type FBot struct {
	ctx context.Context

	// Telegram
	botApi    *tgbotapi.BotAPI
	tgbotCmds []tgbotapi.BotCommand
	msgMap    map[int]tgbotapi.Message

	// OpenAI
	openaiClient   *openai.Client
	aiModels       []openai.Model
	modelMaxTokens int
	selectedModel  string
}

const (
	defaultModel     = openai.GPT3Dot5Turbo
	defaultMaxTokens = 1024

	defaultUpdatesTimeout = 60
)

func NewFBot(cfg *Config) (*FBot, error) {
	if cfg.OpenAI.ApiKey == "" {
		return nil, fmt.Errorf("openAI API key is required")
	}
	if cfg.Telegram.BotApiToken == "" {
		return nil, fmt.Errorf("telegram bot API token is required")
	}

	ctx := context.Background()
	fbot := &FBot{
		ctx:            ctx,
		msgMap:         map[int]tgbotapi.Message{},
		selectedModel:  defaultModel,
		modelMaxTokens: defaultMaxTokens,
	}

	if err := fbot.initTelegram(cfg.Telegram.BotApiToken); err != nil {
		return nil, fmt.Errorf("initializing Telegram failed: %w", err)
	}

	if err := fbot.initOpenAI(cfg.OpenAI.ApiKey); err != nil {
		return nil, fmt.Errorf("initializing OpenAI failed: %w", err)
	}

	fbot.sendBootMessage()

	return fbot, nil
}

const bootChatID = 911111537

func (fbot *FBot) sendBootMessage() {
	if err := fbot.sendTelegramMessage(bootChatID, "Reporting for duty!"); err != nil {
		logrus.Errorf("failed to send boot messaage")
	}
}
func (fbot *FBot) initOpenAI(apiKey string) error {
	// Create a new OpenAI client using the provided API key
	client := openai.NewClient(apiKey)
	fbot.openaiClient = client

	if err := fbot.loadOaModels(); err != nil {
		return err
	}
	logrus.Debugf("OpenAI client activated: %d models", len(fbot.aiModels))

	return nil
}

func (fbot *FBot) loadOaModels() error {
	listResp, err := fbot.openaiClient.ListModels(fbot.ctx)
	if err != nil {
		return fmt.Errorf("OpenAI client error: %v", err)
	}
	logrus.Infof("OpenAI client active, has %d models available", len(listResp.Models))

	for i, model := range listResp.Models {
		idx := i + 1
		logrus.Tracef(" - %3d - %v %v (ownedBy: %v) %+v", idx, model.Object, model.ID, model.OwnedBy, model.Permission)
	}
	fbot.aiModels = listResp.Models
	return nil
}

func (fbot *FBot) initTelegram(botToken string) error {
	// Create a new Telegram bot using the provided API token
	botApi, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return err
	}
	if logrus.GetLevel() >= logrus.DebugLevel {
		botApi.Debug = true
	}

	fbot.botApi = botApi

	if err := fbot.loadTgCommands(); err != nil {
		return err
	}
	logrus.Debugf("Telegram BotAPI activated: %+v", botApi.Self)
	return nil
}

func (fbot *FBot) loadTgCommands() error {
	botCmds, err := fbot.botApi.GetMyCommands()
	if err != nil {
		return fmt.Errorf("bot.GetMyCommands error: %w", err)
	}
	if len(botCmds) > 0 {
		logrus.Debugf("telegram bot has %d commands:", len(botCmds))
		for i, botCmd := range botCmds {
			index := i + 1
			logrus.Debugf(" - command %2d: %v", index, botCmd)
		}
		fbot.tgbotCmds = botCmds
	} else {
		logrus.Debugf("telegram bot has no commands defined")
	}
	return nil
}

func (fbot *FBot) processUpdates() {
	ctx := fbot.ctx
	offset := 0

	updates := fbot.botApi.GetUpdatesChan(tgbotapi.UpdateConfig{
		Offset:  offset,
		Timeout: defaultUpdatesTimeout,
	})

	fmt.Println()
	logrus.Infoln("listening for incoming Telegram updates..")
	defer logrus.Infoln("done listening for incoming Telegram updates")

	// Handle incoming updates from Telegram
	for update := range updates {
		ctx, cancel := context.WithCancel(ctx)
		_ = ctx

		if err := fbot.handleUpdate(update); err != nil {
			logrus.Errorf("handling Telegram update failed: %v", err)
		}

		cancel()
		log.Println("waiting for next Telegram update..")
	}
}

func (fbot *FBot) handleUpdate(update tgbotapi.Update) error {
	sep := strings.Repeat("-", 80)
	logrus.Tracef("INCOMING UPDATE:\n%s\n%s\n%s\n", sep, toJson(update), sep)

	updateMsg := update.Message
	sentFrom := update.SentFrom()
	fromChat := update.FromChat()

	if fromChat == nil {
		logrus.Debugf("update has FromChat=nil")
		return nil
	}

	sentFromType := fmt.Sprint(sentFrom.ID)
	if sentFrom.IsBot {
		sentFromType = fmt.Sprintf("%v (BOT)", sentFromType)
	}

	logrus.Tracef("telegram update (%v) in chat %v (ID: %v, type: %v) from user: %v (%v)", update.UpdateID, fromChat.Title, fromChat.ID, fromChat.Type, sentFrom, sentFromType)

	if updateMsg == nil {
		logrus.Debugf("message is nil..")
		return nil
	}

	// Check if the message is from a user with permission to use the bot
	if !fbot.IsUserAllowed(sentFrom) {
		logrus.Warnf("message sent by user %v not in the list of allowed users", update.Message.From)
		return nil
	}
	// Check if message is for this bot or a reply to previous message from bot
	if !fromChat.IsPrivate() {
		if !fbot.IsMessageForMe(updateMsg) {
			logrus.Debugf("this message is not for me!\n%v\n", toJson(updateMsg))
			return nil
		}
	}

	fbot.msgMap[updateMsg.MessageID] = *updateMsg

	if updateMsg.IsCommand() {
		logrus.Debugf("this message is a command!")
		return nil
	}

	var msgTxt string

	if update.Message.Photo != nil {
		photo := update.Message.Photo
		photoFileID := photo[len(photo)-1].FileID

		logrus.Debugf("message sent with %d images", len(photo))

		fileURL, err := fbot.botApi.GetFileDirectURL(photoFileID)
		if err != nil {
			logrus.Warnf("failed to get image file URL: %v", err)
			return nil
		}

		logrus.Debugf("file URL for file %v: %v", photoFileID, fileURL)

		file, err := downloadImageFromURL(fileURL)
		if err != nil {
			logrus.Warnf("failed to download image file: %v", err)
			return nil
		}

		logrus.Debugf("image downloaded to: %v", file)

		detText, err := detectTextFromImage(file)
		if err != nil {
			logrus.Warnf("failed to detect text in image file: %v", err)
			return nil
		}

		var text string
		if detText == "" {
			text = "No text detected in image."
			logrus.Debugf("no text detected in image")
		} else {
			text = fmt.Sprintf("Here's text detected in image:\n\n%s", detText)
			logrus.Debugf("detected text in image: %v", detText)
		}

		if err := fbot.sendTelegramMessageReply(update.Message.Chat.ID, updateMsg.MessageID, text); err != nil {
			return fmt.Errorf("sending telegram message failed: %v", err)
		}

		msgTxt = fmt.Sprintf("Extract date from the following text extracted from an image and create a google calendar links for creating new event:\n\n%s", detText)
	} else {
		msgTxt = updateMsg.Text
		if msgTxt == "" {
			logrus.Debugf("message text is empty..")
			return nil
		}
	}

	chatMessages := fbot.convertTelegramMsgChainIntoChatCompletionMessages(updateMsg)
	if chatMessages != nil {
		logrus.Debugf("message chain:\n%v\n", toJson(chatMessages))
	}

	logrus.Debugf("sending AI chat completion request: %s", msgTxt)

	chatMessages = append(chatMessages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: msgTxt})

	respMsg, err := fbot.sendAIChatRequest(chatMessages)
	if err != nil {
		return fmt.Errorf("sending AI chat request failed: %w", err)
	}

	if err := fbot.sendTelegramMessageReply(fromChat.ID, updateMsg.MessageID, respMsg); err != nil {
		return fmt.Errorf("sending telegram message failed: %w", err)
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

func (fbot *FBot) sendTelegramMessageReply(chatID int64, replyToMsg int, text string) error {
	// Send the generated response back to the user through the bot
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = replyToMsg

	sent, err := fbot.botApi.Send(msg)
	if err != nil {
		return fmt.Errorf("sending telegram message reply failed: %v", err)
	}

	fbot.msgMap[sent.MessageID] = sent

	return nil
}

func (fbot *FBot) sendTelegramMessage(chatID int64, text string) error {
	// Send the generated response back to the user through the bot
	msg := tgbotapi.NewMessage(chatID, text)
	fbot.botApi.GetChat(tgbotapi.ChatInfoConfig{tgbotapi.ChatConfig{
		ChatID:             0,
		SuperGroupUsername: "",
	}})
	if _, err := fbot.botApi.Send(msg); err != nil {
		return fmt.Errorf("sending telegram message failed: %v", err)
	}

	return nil
}

func (fbot *FBot) IsMessageForMe(msg *tgbotapi.Message) bool {
	if fbot.botApi.IsMessageToMe(*msg) {
		return true
	}
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil && msg.ReplyToMessage.From.ID == fbot.botApi.Self.ID {
		return true
	}
	return false
}

func (fbot *FBot) IsUserAllowed(user *tgbotapi.User) bool {
	if isAllowedUser(user.UserName) {
		return true
	}
	return false
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

func (fbot *FBot) convertTelegramMsgChainIntoChatCompletionMessages(msg *tgbotapi.Message) []openai.ChatCompletionMessage {
	var msgList []tgbotapi.Message
	replyToMsg := fbot.getReplyToMsgFor(msg)
	for replyToMsg != nil {
		msgList = append(msgList, *replyToMsg)
		replyToMsg = fbot.getReplyToMsgFor(replyToMsg)
	}
	var msgs []openai.ChatCompletionMessage
	for i := len(msgList) - 1; i >= 0; i-- {
		msg := msgList[i]
		role := openai.ChatMessageRoleUser
		if msg.From.ID == fbot.botApi.Self.ID {
			role = openai.ChatMessageRoleAssistant
		}
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Text,
		})
	}
	return msgs
}

func (fbot *FBot) getReplyToMsgFor(msg *tgbotapi.Message) *tgbotapi.Message {
	if msg == nil || msg.ReplyToMessage == nil {
		return nil
	}
	replyToMsg, ok := fbot.msgMap[msg.ReplyToMessage.MessageID]
	if !ok || replyToMsg.MessageID == 0 {
		return nil
	}
	return &replyToMsg
}

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
