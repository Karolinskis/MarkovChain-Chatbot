using System;
using TwitchLib.Client;
using TwitchLib.Client.Events;
using TwitchLib.Client.Models;

namespace MarkovChainChatbot;

class Program
{
    static void Main(string[] args)
    {
        string settingsPath = "settings.json";

        Settings.Instance.LoadSettings(settingsPath);

        Database database = new Database($"{Settings.Instance.BotUsername}_{Settings.Instance.ChannelName}_markovchain.db");
        MarkovChainGenerator markovChain = new MarkovChainGenerator(database);

        var chatbot = new Chatbot(Settings.Instance.BotUsername, Settings.Instance.AccessToken, Settings.Instance.ChannelName, markovChain);

        while (!Settings.Instance.TrainingMode)
        {
            System.Threading.Thread.Sleep(Settings.Instance.AutoGenerateInterval * 1000);
            var message = markovChain.GenerateMessage();
            Console.WriteLine($"Generated message: {message}");
            chatbot.SendMessage(message);
        }

        Console.WriteLine("Press any key to exit...");
        Console.ReadKey();
    }
}
