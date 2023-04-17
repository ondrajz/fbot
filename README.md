Fbot
====

FBot is our family assistant chatbot.

## Features

- uses Telegram API for interacting with users
- uses OpenAI API for prompting models

- can convert screenshots containing a date into URL to create event in Google Calendar
- can pass prompts and responses between user and GPT models  

## Quick Start

First, configure auth info:

```sh
export FBOT_TELEGRAM_BOT_API_TOKEN="<TELEGRAM_BOT_API_TOKEN>"
export FBOT_OPENAI_API_KEY="<OPENAI_API_KEY>"
```

To start bot, run:

```sh
go run ./cmd/fbot
```
