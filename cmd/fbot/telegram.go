package main

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

const (
	bootMessageText = `🆙`
)

func (fbot *FBot) initTelegram(botToken string) error {
	// Create a new Telegram bot using the provided API token
	botApi, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return err
	}
	if fbot.isDebugOnFor("telegram") {
		botApi.Debug = true
	}

	fbot.botApi = botApi

	if user, err := fbot.botApi.GetMe(); err != nil {
		logrus.Debugf("GetMe error: %v", err)
	} else {
		logrus.Debugf("GetMe: %v", toJson(user))
	}

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
			logrus.Debugf(" - %2d. command '%s': %v", index, botCmd.Command, botCmd.Description)
		}
		fbot.tgbotCmds = botCmds
	} else {
		logrus.Debugf("telegram bot has no commands defined")
	}
	return nil
}

func (fbot *FBot) sendControlMessage(msg string) error {
	chat, err := fbot.botApi.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{
		ChatID: controlChatId,
		// SuperGroupUsername: "", // Use @ondrajz ?
	}})
	if err != nil {
		return fmt.Errorf("getting control chat failed: %w", err)
	}

	bootMsg := tgbotapi.NewMessage(chat.ID, msg)
	if err := fbot.sendTelegramMessage(bootMsg); err != nil {
		return fmt.Errorf("sending telegram message failed: %w", err)
	}

	return nil
}

/*func (fbot *FBot) sendBootMessage() error {
	chat, err := fbot.botApi.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{
		ChatID: controlChatId,
		// SuperGroupUsername: "", // Use @ondrajz ?
	}})
	if err != nil {
		return fmt.Errorf("getting boot chat failed: %w", err)
	}

	bootMsg := tgbotapi.NewMessage(chat.ID, bootMessageText)
	if err := fbot.sendTelegramMessage(bootMsg); err != nil {
		return fmt.Errorf("sending telegram message failed: %w", err)
	}

	return nil
}*/

func (fbot *FBot) sendTelegramMessageReply(chatID int64, replyToMsg int, text string) error {
	// Send the generated response back to the user through the bot
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = replyToMsg

	err := fbot.sendTelegramMessage(msg)
	if err != nil {
		return fmt.Errorf("send reply error: %w", err)
	}

	return nil
}

func (fbot *FBot) sendTelegramMessage(sendMsg tgbotapi.MessageConfig) error {
	logrus.Tracef("sending telegram message (%v bytes) to chat %v", len(sendMsg.Text), sendMsg.ChatID)

	msg, err := fbot.botApi.Send(sendMsg)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	fbot.msgMap[msg.MessageID] = msg
	logrus.Tracef("telegram message sent successfully: %+v", msg)

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
