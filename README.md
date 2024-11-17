# Markov chain chatbot

This is a simple Twitch chatbot that uses a Markov chain to generate responses based on what it learned from Twitch chat. It is heavely inspired by [TwitchMarkovChain](https://github.com/tomaarsen/TwitchMarkovChain).

## Configuration

The bot is configured using a `settings.json` file. The file should look like this

```json
{
  "BotUsername": "botUsername",
  "AccessToken": "accessToken",
  "ChannelName": "channelName",
  "TrainingMode": true,
  "AllowedUsers": ["allowedUser1", "allowedUser2"],
  "BlockedUsers": ["blockedUser1", "blockedUser2"],
  "MinSentenceWords": -1,
  "MaxSentenceWords": 20,
  "AutoGenerateMessages": true,
  "AutoGenerateInterval": 5000,
  "AllowGenerateCommand": true,
  "GenerateCommands": ["!generate"]
}
```

- `BotUsername`: The username of the bot account.
- `AccessToken`: The access token of the bot account.
- `ChannelName`: The name of the channel the bot should join.
- `TrainingMode`: If set to `true`, the bot will only learn from chat messages, but not generate any messages. If set to `false`, the bot will learn from chat messages and generate messages.
- `AllowedUsers`: A list of users that are allowed to use the bot.
- `BlockedUsers`: A list of users that the chatbot will ignore when learning from chat messages.
- `MinSentenceWords`: The minimum amount of words a generated sentence should have.
- `MaxSentenceWords`: The maximum amount of words a generated sentence should have.
- `AutoGenerateMessages`: If set to `true`, the bot will automatically generate messages.
- `AutoGenerateInterval`: The interval in seconds between automatically generated messages.
- `AllowGenerateCommand`: If set to `true`, users can generate messages using the commands in `GenerateCommands`.
- `GenerateCommands`: A list of commands that users can use to generate messages.

## Contributing

If you want to contribute to this project, feel free to create a pull request. If you have any questions or suggestions, feel free to open an issue.
