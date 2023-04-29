FBot
====

FBot is our assistant chatbot.

## Features

- can convert screenshots containing a date into URL to create event in Google Calendar
- can pass prompts and responses between user and GPT models  

## Third-Party Dependencies

- Telegram BotAPI for user interaction
- OpenAI API for AI capabilities
- Gosseract for text detection in images

## Quick Start

Setup authentication:

```sh
# Setup authentication 
export FBOT_TELEGRAM_BOT_API_TOKEN="<TELEGRAM_BOT_API_TOKEN>"
export FBOT_OPENAI_API_KEY="<OPENAI_API_KEY>"

# Install & Run
go run ./cmd/fbot
```
