# Auto-Delete Telegram Bot

This is an auto-delete Telegram bot built using Go, MongoDB, and the Telegram Bot API. The bot allows users to set auto-delete timers for messages in groups and tracks the users and settings in a MongoDB database.

## Features

- Add users to the MongoDB database upon starting the bot.
- Set auto-delete timers for messages in groups.
- Stop the auto-delete timer for groups.
- Fetch bot statistics (number of users and groups).
- Support for multiple time units: seconds (`s`), minutes (`m`), hours (`h`), and days (`d`).

## Requirements

### Fill all requirements into [main.go] (https://github.com/PrinceStarLord/go-autodelete-bot/blob/main/main.go)

- Go (version 1.18+)
- MongoDB 
- Telegram Bot Token (created via [BotFather](https://core.telegram.org/bots#botfather))

## Setup

### 1. Clone the repository:

```bash
git clone https://github.com/PrinceStarLord/go-autodelete-bot
cd go-autodelete-bot
