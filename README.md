# Markov Chain Chatbot

A Twitch chatbot that uses a Markov chain to generate messages based on what it learns from chat. Inspired by [TwitchMarkovChain](https://github.com/tomaarsen/TwitchMarkovChain).

## Features

- Learns from Twitch chat messages in realtime
- Generates messages using a second order Markov chain
- Configurable auto posting on a timer
- Chat commands for generation (`!generate`, etc.)
- Blacklisted word filtering with Unicode normalization
- Discord webhook notifications for generated messages
- SQLite storage (pure Go, no CGO required)
- Training-only mode for building a dataset without posting

## Requirements

- Go 1.26+
- A Twitch account for the bot with an OAuth token

## Build & Run

```bash
go build -o markovchain-chatbot
./markovchain-chatbot settings.json
```

The settings file path defaults to `settings.json` if not provided.

## Configuration

Create a `settings.json` file:

```json
{
  "BotUsername": "botUsername",
  "AccessToken": "oauth:your_token_here",
  "ChannelName": "channelName",
  "TrainingMode": false,
  "AllowedUsers": ["*"],
  "BlockedUsers": ["nightbot", "streamelements"],
  "MinSentenceWords": -1,
  "MaxSentenceWords": 25,
  "AutoGenerateMessages": true,
  "AutoGenerateInterval": 180,
  "AllowGenerateCommand": true,
  "GenerateCommands": ["!generate"],
  "BlacklistedWords": [],
  "EnableDiscordLogging": false,
  "DiscordWebhookUrl": "",
  "AllowNonAsciiMessages": false
}
```

| Field | Description |
|-------|-------------|
| `BotUsername` | Twitch username of the bot account |
| `AccessToken` | OAuth token for the bot account |
| `ChannelName` | Twitch channel to join |
| `TrainingMode` | If `true`, only learns — never posts messages |
| `AllowedUsers` | Users allowed to use commands (`*` = everyone) |
| `BlockedUsers` | Users ignored for training (bots, etc.) |
| `MinSentenceWords` | Minimum words in a generated sentence (`-1` = no limit) |
| `MaxSentenceWords` | Maximum words in a generated sentence |
| `AutoGenerateMessages` | Automatically post messages on a timer |
| `AutoGenerateInterval` | Seconds between auto-generated messages |
| `AllowGenerateCommand` | Allow chat commands to trigger generation |
| `GenerateCommands` | Commands that trigger generation |
| `BlacklistedWords` | Words that will never appear in generated messages |
| `EnableDiscordLogging` | Forward generated messages to a Discord webhook |
| `DiscordWebhookUrl` | Discord webhook URL |
| `AllowNonAsciiMessages` | Allow non-ASCII characters in generated messages |

## Chat Commands

| Command | Description |
|---------|-------------|
| `!stats` | Shows dataset statistics (start pairs, grammar entries) |
| Custom generate commands | Generates and replies with a Markov chain message |

## Project Structure

```
├── main.go           # Entry point
├── chatbot/          # Twitch IRC client and message handling
├── markov/           # Markov chain training and generation
├── database/         # SQLite persistence layer
├── tokenizer/        # Sentence tokenization and detokenization
├── filter/           # Message filtering (links, mentions, commands)
├── discord/          # Discord webhook notifications
└── settings/         # JSON config loading
```

## Contributing

If you want to contribute to this project, feel free to create a pull request. If you have any questions or suggestions, feel free to open an issue.
