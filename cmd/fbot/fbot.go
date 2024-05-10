package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gookit/color"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

const (
	defaultUpdatesTimeout = 60
	controlChatId         = 911111537

	createEventPrompt = `Create URL link for creating a new event in Google calendar using the following information`
)

type FBot struct {
	ctx context.Context

	debug string

	// Telegram
	botApi    *tgbotapi.BotAPI
	tgbotCmds []tgbotapi.BotCommand
	msgMap    map[int]tgbotapi.Message

	// OpenAI
	openaiClient   *openai.Client
	aiModels       []openai.Model
	modelMaxTokens int
	selectedModel  string
	started        time.Time
}

func NewFBot(cfg *Config) (*FBot, error) {
	fbot := &FBot{
		ctx:            context.Background(),
		debug:          cfg.Debug,
		msgMap:         map[int]tgbotapi.Message{},
		selectedModel:  defaultModel,
		modelMaxTokens: defaultMaxTokens,
		started:        time.Now(),
	}

	if err := fbot.initTelegram(cfg.Telegram.BotApiToken); err != nil {
		return nil, fmt.Errorf("initializing Telegram failed: %w", err)
	}

	if err := fbot.initOpenAI(cfg.OpenAI.ApiKey); err != nil {
		return nil, fmt.Errorf("initializing OpenAI failed: %w", err)
	}

	if err := fbot.sendControlMessage(bootMessageText); err != nil {
		logrus.Warnf("sending boot message failed: %v", err)
	}

	return fbot, nil
}

func (fbot *FBot) isDebugOnFor(unit string) bool {
	if fbot.debug == "" {
		return false
	}
	return IsDebugEnabled(fbot.debug, unit)
}

func (fbot *FBot) traceLogf(unit string, format string, args ...interface{}) {
	if fbot.isDebugOnFor(unit) {
		logrus.Tracef(unit+": "+format, args...)
	}
}

func (fbot *FBot) processUpdates() {
	ctx := fbot.ctx
	offset := 0

	updates := fbot.botApi.GetUpdatesChan(tgbotapi.UpdateConfig{
		Offset:         offset,
		Limit:          0,
		Timeout:        defaultUpdatesTimeout,
		AllowedUpdates: nil,
	})

	logrus.WithField("len", len(updates)).Infoln("begin processing of incoming Telegram updates..")
	defer logrus.Infoln("done processing of incoming Telegram updates")

	// Handle incoming updates from Telegram
	for {
		log.Println("waiting for next Telegram update..")

		select {
		case update := <-updates:
			ctx, cancel := context.WithCancel(ctx)
			_ = ctx

			if err := fbot.processUpdate(update); err != nil {
				logrus.Errorf("handling Telegram update failed: %v", err)
			}

			cancel()
		}

	}
}

func (fbot *FBot) processUpdate(update tgbotapi.Update) error {
	updateMsg := update.Message
	sentFrom := update.SentFrom()
	fromChat := update.FromChat()

	var sentFromType string
	if sentFrom.IsBot {
		sentFromType = "BOT"
	} else {
		sentFromType = "USER"
	}

	sep := strings.Repeat("-", 80)
	logrus.Tracef("%s\nTelegram UPDATE:\n%s\n%s\n%s\n", sep, sep, color.Gray.Sprint(toJson(update)), sep)

	if fromChat == nil {
		logrus.Tracef("update.FromChat==nil, ignoring update")
		return nil
	}
	if updateMsg == nil {
		logrus.Tracef("update.Message==nil, ignoring update")
		return nil
	}
	logrus.Debugf("processing Telegram UPDATE #%v | IN: %s [%v]: %v | FROM: %s[%v]: %v", update.UpdateID, fromChat.Type, fromChat.ID, fromChat.Title, sentFromType, sentFrom.ID, sentFrom)

	// Check if the message is from a user with permission to use the bot
	if !fbot.IsUserAllowed(sentFrom) {
		logrus.Warnf("the message from user %v not in the allowed users, ignoring update", update.Message.From)
		return nil
	}

	// Check if message is for this bot or a reply to previous message from bot
	if !fromChat.IsPrivate() {
		if updateMsg.Photo == nil && !fbot.IsMessageForMe(updateMsg) {
			logrus.Debugf("this message is not for me!\n%v\n", toJson(updateMsg))
			return nil
		}
	}

	// store this message for future reference
	fbot.msgMap[updateMsg.MessageID] = *updateMsg

	/*if updateMsg.IsCommand() {
		return fbot.processCommandMessage(updateMsg)
	}

	return nil*/

	switch {
	case updateMsg.IsCommand():
		return fbot.processCommandMessage(updateMsg)
	default:
		return fbot.processOtherMessage(updateMsg)
	}
}

func (fbot *FBot) processOtherMessage(msg *tgbotapi.Message) error {
	logrus.Debugf("processing other message: %v", msg.Text)

	var msgTxt string

	if msg.Photo != nil {
		photo := msg.Photo
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

		if err := fbot.sendTelegramMessageReply(msg.Chat.ID, msg.MessageID, text); err != nil {
			return fmt.Errorf("sending telegram message failed: %v", err)
		}

		msgTxt = fmt.Sprintf("Extract date from the following text extracted from an image and create a google calendar links for creating new event:\n\n%s", detText)
	} else {
		msgTxt = msg.Text
		if msgTxt == "" {
			logrus.Debugf("message text is empty..")
			return nil
		}
	}

	chatMessages := fbot.convertTelegramMsgChainIntoChatCompletionMessages(msg)
	if chatMessages != nil {
		logrus.Tracef("message chain:\n%v\n", color.Gray.Sprint(toJson(chatMessages)))
	}
	chatMessages = append(chatMessages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: msgTxt})

	logrus.Debugf("sending AI chat completion request: %s", msgTxt)

	reply, err := fbot.sendAIChatRequest(chatMessages)
	if err != nil {
		return fmt.Errorf("sending AI chat request failed: %w", err)
	}

	if err := fbot.sendTelegramMessageReply(msg.Chat.ID, msg.MessageID, reply); err != nil {
		return fmt.Errorf("sending telegram message failed: %w", err)
	}

	return nil
}

func (fbot *FBot) processCommandMessage(msg *tgbotapi.Message) error {
	logrus.Debugf("processing command: %v", msg.Command())

	switch strings.ToLower(msg.Command()) {
	case "status":
		if err := fbot.commandStatus(msg); err != nil {
			return fmt.Errorf("command 'status' failed: %w", err)
		}
	case "event":
		if err := fbot.commandEvent(msg); err != nil {
			return fmt.Errorf("command 'event' failed: %w", err)
		}
	default:
		logrus.Warnf("unknown command: %v", msg.Command())
		msgText := fmt.Sprintf("Unknown command '%s'", msg.Command())
		if err := fbot.sendTelegramMessageReply(msg.Chat.ID, msg.MessageID, msgText); err != nil {
			return fmt.Errorf("sending telegram message failed: %w", err)
		}
	}

	return nil
}

func (fbot *FBot) commandStatus(msg *tgbotapi.Message) error {
	respMsg := fmt.Sprintf("FBot is online for %v\nsince: %v", time.Since(fbot.started).Round(time.Second), fbot.started.Format(time.UnixDate))

	if err := fbot.sendTelegramMessageReply(msg.Chat.ID, msg.MessageID, respMsg); err != nil {
		return fmt.Errorf("sending telegram message failed: %w", err)
	}

	return nil
}

func (fbot *FBot) commandEvent(msg *tgbotapi.Message) error {
	chatMessages := newUserCompletionMessage(fmt.Sprintf("%s:\n\n```\n%s\n```", createEventPrompt, msg.CommandArguments()))

	respMsg, err := fbot.sendAIChatRequest(chatMessages)
	if err != nil {
		m := fmt.Sprintf("Sorry, AI has failed:\n%s", err.Error())
		if err := fbot.sendTelegramMessageReply(msg.Chat.ID, msg.MessageID, m); err != nil {
			logrus.Warnf("sending telegram error message failed: %v", err)
		}
		return fmt.Errorf("sending AI chat request failed: %w", err)
	}

	if err := fbot.sendTelegramMessageReply(msg.Chat.ID, msg.MessageID, respMsg); err != nil {
		return fmt.Errorf("sending telegram message failed: %w", err)
	}

	return nil
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
