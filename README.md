# Markov Chain Chatbot

[![CI](https://github.com/Karolinskis/MarkovChain-Chatbot/actions/workflows/ci.yml/badge.svg)](https://github.com/Karolinskis/MarkovChain-Chatbot/actions/workflows/ci.yml)

A Twitch chatbot that learns from chat in realtime and generates messages using a second-order Markov chain. Inspired by [TwitchMarkovChain](https://github.com/tomaarsen/TwitchMarkovChain).

## Features

- Learns from Twitch chat messages in realtime
- Generates messages using a second-order Markov chain
- Runs multiple bot accounts, each in multiple channels, from one process
- Stream live detection via the Twitch Helix API — only learns and auto-posts while the stream is live
- Untrains deleted messages: when moderators delete a message, its contribution to the chain (and that of all replies to it) is removed
- Configurable auto-posting on a timer
- Chat commands for generation (`!generate`, etc.) with a per-channel user allowlist
- Blacklisted word filtering with Unicode normalization, plus link/mention/command filtering of generated output
- PostgreSQL storage, partitioned per channel

## Requirements

- Go 1.26+
- PostgreSQL
- A Twitch account for the bot with an OAuth token ([twitchtokengenerator.com](https://twitchtokengenerator.com) works)
- Optional: a Twitch application (client ID + secret) for live detection — without it, every channel is treated as always live

## Build & Run

```bash
# 1. Apply database migrations
go run ./cmd/migrate -settings settings.json up

# 2. Run the bot
go build -o markovchain-chatbot ./cmd/bot
./markovchain-chatbot settings.json
```

The settings file path defaults to `settings.json` if not provided.

## Configuration

```json
{
  "DatabaseURL": "postgres://user:password@localhost:5432/markovbot",
  "HelixClientID": "your_twitch_app_client_id",
  "HelixClientSecret": "your_twitch_app_client_secret",
  "Bots": [
    {
      "BotUsername": "botUsername",
      "AccessToken": "oauth:your_token_here",
      "Channels": [
        {
          "ChannelName": "channelName",
          "TrainingMode": false,
          "AllowedUsers": ["*"],
          "BlockedUsers": ["nightbot", "streamelements"],
          "MaxSentenceWords": 25,
          "AutoGenerateMessages": true,
          "AutoGenerateInterval": 180,
          "AllowGenerateCommand": true,
          "GenerateCommands": ["!generate"],
          "BlacklistedWords": [],
          "AllowNonAsciiMessages": false
        }
      ]
    }
  ]
}
```

### Top level

| Field | Description |
|-------|-------------|
| `DatabaseURL` | PostgreSQL connection string |
| `HelixClientID` | Twitch application client ID (optional, enables live detection) |
| `HelixClientSecret` | Twitch application client secret |
| `Bots` | List of bot accounts to run |

### Per bot

| Field | Description |
|-------|-------------|
| `BotUsername` | Twitch username of the bot account |
| `AccessToken` | OAuth token for the bot account (`oauth:` prefix optional) |
| `Channels` | List of channels this bot joins |

### Per channel

| Field | Description |
|-------|-------------|
| `ChannelName` | Twitch channel to join |
| `TrainingMode` | If `true`, disables the auto-post timer — the bot only learns |
| `AllowedUsers` | Users allowed to use generate commands (`*` = everyone) |
| `BlockedUsers` | Users ignored for both training and commands (other bots, etc.) |
| `MaxSentenceWords` | Maximum words in a generated sentence |
| `AutoGenerateMessages` | Automatically post messages on a timer |
| `AutoGenerateInterval` | Seconds between auto-generated messages |
| `AllowGenerateCommand` | Allow chat commands to trigger generation |
| `GenerateCommands` | Command prefixes that trigger generation |
| `BlacklistedWords` | Words that must never appear in generated messages |
| `AllowNonAsciiMessages` | Allow non-ASCII characters in generated messages |

## Metrics

The bot serves Prometheus metrics on `/metrics` (default `:9091`, override with the `METRICS_ADDR` env var): training/generation/untrain counts and errors per channel, live status per channel, and IRC connection state per bot account.

## Chat Commands

| Command | Description |
|---------|-------------|
| `!stats` | Shows dataset statistics (start pairs, grammar entries) |
| Generate commands | Generates and replies with a Markov chain message |

## Docker

The compose file runs the bot and expects PostgreSQL to be reachable on an external `backend` network. It mounts `./settings.json` read-only.

```bash
docker compose run --rm migrate   # apply database migrations
docker compose up -d bot          # start the bot
```

## Migrating from SQLite

Older versions stored each channel in a SQLite file. `cmd/migrate-sqlite` imports one into PostgreSQL:

```bash
go run ./cmd/migrate-sqlite \
  -sqlite old_markovchain.db \
  -postgres "postgres://user:password@localhost:5432/markovbot" \
  -channel channelName \
  -bot botUsername
```

## Project Structure

```
├── cmd/
│   ├── bot/            # Bot entry point
│   ├── migrate/        # Database migration runner (goose)
│   └── migrate-sqlite/ # One-off SQLite → PostgreSQL importer
└── internal/
    ├── chatbot/        # Twitch IRC client, channels, live poller
    ├── markov/         # Markov chain training and generation
    ├── database/       # PostgreSQL persistence layer and migrations
    ├── helix/          # Twitch Helix API client (live detection)
    ├── tokenizer/      # Sentence tokenization and detokenization
    ├── filter/         # Message filtering (links, mentions, commands)
    └── settings/       # JSON config loading
```

## Contributing

If you want to contribute to this project, feel free to create a pull request. If you have any questions or suggestions, feel free to open an issue.
