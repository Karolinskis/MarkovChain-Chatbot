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

        Settings settings = new Settings();

        settings.LoadSettings(settingsPath);

        Database database = new Database($"{settings.BotUsername}_markovchain.db");
        MarkovChainGenerator markovChain = new MarkovChainGenerator(database);

        var chatbot = new Chatbot(settings.BotUsername, settings.AccessToken, settings.ChannelName, markovChain);

        while (true)
        {
            System.Threading.Thread.Sleep(settings.AutoGenerateInterval * 1000);
            var message = markovChain.GenerateMessage();
            Console.WriteLine($"Generated message: {message}");
            chatbot.SendMessage(message);
        }

        Console.WriteLine("Press any key to exit...");
        Console.ReadKey();
    }
}
