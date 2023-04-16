package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/otiai10/gosseract/v2"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	//	"google.golang.org/api/vision/v1"
)

var (
	openAiApiKey   = os.Getenv("YOUR_OPENAI_API_KEY")
	telegramApiKey = os.Getenv("YOUR_TELEGRAM_BOT_API_TOKEN")
)

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}

func main() {
	ctx := context.Background()

	// Create a new Telegram bot using the provided API token
	bot, err := tgbotapi.NewBotAPI(telegramApiKey)
	if err != nil {
		logrus.Fatalf("Telegram bot error: %v", err)
	} else {
		logrus.Infof("Telegram BotAPI active: %+v", bot.Self)
	}

	botCmds, err := bot.GetMyCommands()
	if err != nil {
		logrus.Errorf("bot.GetMyCommands error: %v", err)
	} else {
		if len(botCmds) > 0 {
			logrus.Debugf("listing %d bot commands:", len(botCmds))
			for i, botCmd := range botCmds {
				index := i + 1
				logrus.Debugf(" - bot command [%d]: %v", index, botCmd)
			}
		}
	}

	// Create a new OpenAI client using the provided API key
	client := openai.NewClient(openAiApiKey)

	listResp, err := client.ListModels(ctx)
	if err != nil {
		logrus.Fatalf("OpenAI client error: %v", err)
	} else {
		logrus.Infof("OpenAI client active, listing %d models:", len(listResp.Models))
		for _, model := range listResp.Models {
			logrus.Debugf(" - %v [%v]: %+v", model.Object, model.ID, model.Root)
		}
	}

	// Replace YOUR_PROJECT_ID with your Google Cloud project ID
	// ctx := context.Background()
	/*creds, err := google.FindDefaultCredentials(ctx, vision.CloudPlatformScope)
	  if err != nil {
	  	logrus.Fatalf("FindDefaultCredentials error: %v", err)
	  }
	  conf := &jwt.Config{
	  	Email:      creds.JSON["client_email"].(string),
	  	PrivateKey: creds.JSON["private_key"].([]byte),
	  	Scopes: []string{
	  		vision.CloudPlatformScope,
	  	},
	  	TokenURL: google.JWTTokenURL,
	  }
	  gclient, err := vision.New(conf.Client(ctx))
	  if err != nil {
	  	logrus.Fatalf("vision.New error: %v", err)
	  }*/

	// detectText gets text from the Vision API for an image at the given file path.
	/*detectText := func(file string) (string, error) {
		client, err := vision.NewImageAnnotatorClient(ctx)
		if err != nil {
			return "", err
		}

		f, err := os.Open(file)
		if err != nil {
			return "", err
		}
		defer f.Close()

		image, err := vision.NewImageFromReader(f)
		if err != nil {
			return "", err
		}
		annotations, err := client.DetectTexts(ctx, image, nil, 10)
		if err != nil {
			return "", err
		}

		if len(annotations) == 0 {
			logrus.Debugf("No text found.")
		} else {
			var buf bytes.Buffer
			logrus.Debugf("found %d annotations in image", len(annotations))
			for _, annotation := range annotations {
				fmt.Fprintf(&buf, "%q\n", annotation.Description)
			}
			return buf.String(), nil
		}

		return "", nil
	}*/

	// Set up updates channel to receive incoming messages from the user
	updates := bot.GetUpdatesChan(tgbotapi.UpdateConfig{
		Offset:  0,
		Timeout: 60,
	})

	fmt.Println()
	logrus.Infoln("waiting for updates from Telegram..")

	// Handle incoming messages from the user
	for update := range updates {
		ctx, _ = context.WithCancel(ctx)

		updateMsg := update.Message
		sentFrom := update.SentFrom()
		fromChat := update.FromChat()
		chatID := fromChat.ID
		sentFromType := fmt.Sprint(sentFrom.ID)
		if sentFrom.IsBot {
			sentFromType = fmt.Sprintf("%v (BOT)", sentFromType)
		}

		logrus.Tracef("incoming update (%v) in chat %v (%v) from user: %v (%v)", update.UpdateID, fromChat.Title, fromChat.Type, sentFrom, sentFromType)

		if updateMsg == nil {
			logrus.Debugf("message is nil..")
			continue
		}

		// Check if the message is from a user with permission to use the bot
		if !isUserPermitted(sentFrom) {
			logrus.Warnf("user %v is not permitted..", update.Message.From)
			continue
		}
		if !bot.IsMessageToMe(*updateMsg) {
			logrus.Warnf("message is not for me!")
		}

		var msgTxt string

		if update.Message.Photo != nil {
			photo := update.Message.Photo
			photoFileID := photo[len(photo)-1].FileID

			logrus.Debugf("message sent with %d images", len(photo))

			fileURL, err := bot.GetFileDirectURL(photoFileID)
			if err != nil {
				logrus.Warnf("failed to get image file URL: %v", err)
				continue
			}

			logrus.Debugf("file URL for file %v: %v", photoFileID, fileURL)

			file, err := downloadImage(fileURL)
			if err != nil {
				logrus.Warnf("failed to download image file: %v", err)
				continue
			}

			logrus.Debugf("image downloaded to: %v", file)

			detText, err := detectText(file)
			if err != nil {
				logrus.Warnf("failed to detect text in image file: %v", err)
				continue
			}

			var text string
			if detText == "" {
				text = "No text detected in image."
				logrus.Debugf("no text detected in image")
			} else {
				text = fmt.Sprintf("Here's text detected in image:\n\n%s", detText)
				logrus.Debugf("detected text in image: %v", detText)
			}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			msg.ReplyToMessageID = updateMsg.MessageID
			_, err = bot.Send(msg)
			if err != nil {
				logrus.Errorf("failed to send telegram message: %v", err)
				continue
			}

			msgTxt = fmt.Sprintf("Extract date from the following text extracted from an image and create a google calendar links for creating new event:\n\n%s", detText)
		} else {
			msgTxt = updateMsg.Text
			if msgTxt == "" {
				logrus.Debugf("message is empty..")
				continue
			}
		}

		// log.Printf("user message: %v", update.Message.Text)

		// Use the OpenAI API to generate a response to the user's message
		/*response, err := openaiClient.Completions.Create(
		      &openai.CompletionRequest{
		          Model:     "YOUR_OPENAI_MODEL_NAME",
		          Prompt:    update.Message.Text,
		          MaxTokens: 50,
		      },
		  )
		  if err != nil {
		      log.Panic(err)
		  }*/

		logrus.Debugf("sending chat completion request to OpenAI: %s", msgTxt)

		t0 := time.Now()
		resp, err := client.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: msgTxt,
					},
				},
				MaxTokens: 300,
			},
		)
		if err != nil {
			logrus.Errorf("ChatCompletion error: %v", err)
			continue
		}

		took := time.Since(t0)

		// logrus.Tracef("chatCompletion response (%d choices): %+v", len(resp.Choices), resp)
		respMsg := resp.Choices[0].Message.Content

		logrus.Debugf("chatCompletion response (took %v): %v\n\n", took, respMsg)

		// Send the generated response back to the user through the bot
		msg := tgbotapi.NewMessage(chatID, respMsg)
		msg.ReplyToMessageID = updateMsg.MessageID

		if _, err := bot.Send(msg); err != nil {
			logrus.Errorf("Telegram message Send error: %v\n", err)
			continue
		}

		log.Println("waiting for next Telegram update..")
	}
}

func downloadImage(url string) (string, error) {
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

// Check if the user with the given ID is permitted to use the bot
func isUserPermitted(user *tgbotapi.User) bool {
	if allowedUsername(user.UserName) {
		return true
	}
	return false
}

var allowedUsernames = []string{
	"qwkymbfirwnafnb",
	"ondrajz",
}

func allowedUsername(name string) bool {
	for _, allowed := range allowedUsernames {
		if name == allowed {
			return true
		}
	}
	return false
}

func detectText(file string) (string, error) {
	// ctx := context.Background()
	// ctx, cancel := context.WithCancel(ctx)
	// defer cancel()

	/*client, err := vision.NewImageAnnotatorClient(ctx, option.WithCredentialsFile("/home/ondrej/.config/gcloud/vision.json"))
	  if err != nil {
	  	return "", err
	  }
	  defer client.Close()

	  f, err := os.Open(file)
	  if err != nil {
	  	return "", err
	  }
	  defer f.Close()

	  image, err := vision.NewImageFromReader(f)
	  if err != nil {
	  	return "", err
	  }
	  annotations, err := client.DetectTexts(ctx, image, nil, 10)
	  if err != nil {
	  	return "", err
	  }

	  if len(annotations) == 0 {
	  	logrus.Debugf("No text found.")
	  } else {
	  	var buf bytes.Buffer
	  	logrus.Debugf("found %d annotations in image", len(annotations))
	  	for _, annotation := range annotations {
	  		fmt.Fprintf(&buf, "%q\n", annotation.Description)
	  	}

	  	return buf.String(), nil
	  }*/

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
