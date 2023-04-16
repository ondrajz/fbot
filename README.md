Fbot
====

FBot is our family assistant chatbot.

## Features

- uses Telegram API for interacting with users
- uses OpenAI API for prompting models

- can convert screenshots containing a date into URL to create event in Google Calendar
- can pass prompts and responses between user and GPT models  

## Quick Start


```sh
# Set API keys
export TELEGRAM_API_KEY="<TELEGRAM_API_KEY>"
export OPENAI_API_KEY="<OPENAI_API_KEY>"

# Run
go run .
```