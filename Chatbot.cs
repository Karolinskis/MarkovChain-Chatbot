using MarkovChainChatbot.Utils;
using TwitchLib.Client;
using TwitchLib.Client.Events;
using TwitchLib.Client.Models;
using TwitchLib.Communication.Clients;
using TwitchLib.Communication.Models;

namespace MarkovChainChatbot;

public class Chatbot
{
    private TwitchClient _client;
    private MarkovChainGenerator _markovChain;

    public Chatbot(string botUsername, string accessToken, string channelName, MarkovChainGenerator markovChain)
    {
        _markovChain = markovChain;

        ConnectionCredentials credentials = new ConnectionCredentials(botUsername, accessToken);

        var clientOptions = new ClientOptions
        {
            MessagesAllowedInPeriod = 750,
            ThrottlingPeriod = TimeSpan.FromSeconds(30)
        };

        WebSocketClient customClient = new WebSocketClient(clientOptions);
        _client = new TwitchClient(customClient);
        _client.Initialize(credentials, channelName);

        _client.OnLog += Client_OnLog;
        _client.OnJoinedChannel += Client_OnJoinedChannel;
        _client.OnMessageReceived += Client_OnMessageReceived;
        _client.OnWhisperReceived += Client_OnWhisperReceived;
        _client.OnConnected += Client_OnConnected;

        _client.Connect();
    }

    public void SendMessage(string message)
    {
        if (string.IsNullOrWhiteSpace(message))
        {
            return;
        }

        Logger.Instance.Log($"Sending message: {message}", sendToDiscord: true);
        _client.SendMessage(_client.JoinedChannels[0], message);
    }

    private void Client_OnLog(object? sender, OnLogArgs e)
    {
        //Console.WriteLine($"{e.DateTime.ToString()}: {e.BotUsername} - {e.Data}");
    }

    private void Client_OnConnected(object? sender, OnConnectedArgs e)
    {
        Logger.Instance.Log($"Connected to Twitch", sendToDiscord: false);
    }

    private void Client_OnJoinedChannel(object? sender, OnJoinedChannelArgs e)
    {
        Logger.Instance.Log($"Joined channel {e.Channel}", sendToDiscord: false);
    }

    private void Client_OnMessageReceived(object? sender, OnMessageReceivedArgs e)
    {
        // Ignore messages from the bot itself
        if (e.ChatMessage.Username == _client.TwitchUsername)
            return;

        if (Settings.Instance?.BlockedUsers?.Contains(e.ChatMessage.Username) == true)
            return;

        if (e.ChatMessage.Message.TrimStart().Equals("!stats", StringComparison.OrdinalIgnoreCase))
        {
            var stats = _markovChain.GetStatistics();
            string statsMessage = $"Dataset Statistics: Start Pairs: {stats["TotalStartPairs"]}, Grammar Entries: {stats["TotalGrammarEntries"]}";
            _client.SendReply(_client.JoinedChannels[0], e.ChatMessage.Id, statsMessage);
            return; // Don't train on stats command
        }


        // Check for generate commands
        if (Settings.Instance.AllowGenerateCommand &&
            Settings.Instance.GenerateCommands != null &&
            Settings.Instance.GenerateCommands.Any(cmd =>
                e.ChatMessage.Message.TrimStart().StartsWith(cmd, StringComparison.OrdinalIgnoreCase)))
        {
            // Check if user is allowed to use generate commands
            if (Settings.Instance.AllowedUsers != null &&
            (Settings.Instance.AllowedUsers.Contains("*") ||
             Settings.Instance.AllowedUsers.Contains(e.ChatMessage.Username, StringComparer.OrdinalIgnoreCase)))
            {
                var generatedMessage = _markovChain.GenerateMessage();
                _client.SendReply(_client.JoinedChannels[0], e.ChatMessage.Id, generatedMessage);
            }
            else
            {
                Logger.Instance.Log($"User {e.ChatMessage.Username} attempted to use generate command but is not allowed", sendToDiscord: false);
            }
            return; // Don't train on command messages
        }

        Logger.Instance.Log($"Received message: {e.ChatMessage.Username} - {e.ChatMessage.Message}", sendToDiscord: false);

        List<string> tokens = Tokenizer.Tokenize(e.ChatMessage.Message);

        // Train the Markov Chain with the tokenized message
        _markovChain.Train(tokens);
    }

    private void Client_OnWhisperReceived(object? sender, OnWhisperReceivedArgs e)
    {
        // Ignore whispers
    }
}
