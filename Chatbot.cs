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

        Console.WriteLine($"Sending message: {message}");
        _client.SendMessage(_client.JoinedChannels[0], message);
    }

    private void Client_OnLog(object? sender, OnLogArgs e)
    {
        //Console.WriteLine($"{e.DateTime.ToString()}: {e.BotUsername} - {e.Data}");
    }

    private void Client_OnConnected(object? sender, OnConnectedArgs e)
    {
        Console.WriteLine($"Connected to Twitch");
    }

    private void Client_OnJoinedChannel(object? sender, OnJoinedChannelArgs e)
    {
        Console.WriteLine($"Joined channel {e.Channel}");
    }

    private void Client_OnMessageReceived(object? sender, OnMessageReceivedArgs e)
    {
        // Ignore messages from the bot itself
        if (e.ChatMessage.Username == _client.TwitchUsername)
        {
            return;
        }

        if (Settings.Instance?.BlockedUsers?.Contains(e.ChatMessage.Username) == true)
        {
            return;
        }

        Console.WriteLine($"{e.ChatMessage.Username} - {e.ChatMessage.Message}");

        List<string> tokens = Tokenizer.Tokenize(e.ChatMessage.Message);

        // Train the Markov Chain with the tokenized message
        _markovChain.Train(tokens);
    }

    private void Client_OnWhisperReceived(object? sender, OnWhisperReceivedArgs e)
    {
        // Ignore whispers
    }
}
