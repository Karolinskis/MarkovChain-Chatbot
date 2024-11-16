using System.Text.Json;
using System.Text.Json.Serialization;

public class Settings
{
    public string BotUsername { get; set; }
    public string AccessToken { get; set; }
    public string ChannelName { get; set; }
    public List<string>? AllowedUsers { get; set; }
    public List<string>? BlockedUsers { get; set; }
    public int MinSentenceWords { get; set; }
    public int MaxSentenceWords { get; set; }
    public bool AutoGenerateMessages { get; set; }
    public int AutoGenerateInterval { get; set; }
    public bool AllowGenerateCommand { get; set; }
    public List<string> GenerateCommands { get; set; }

    /// <summary>
    /// Loads settings from the specified path
    /// </summary>
    /// <param name="path">Path to read the settings from</param>
    public void LoadSettings(string path)
    {
        if (!File.Exists(path))
        {
            Console.WriteLine("Settings file not found, generating default settings");
            GenerateDefaultSettings(path);
            return;
        }

        try
        {
            string json = File.ReadAllText(path);
            var settings = JsonSerializer.Deserialize<Settings>(json);

            if (settings == null)
            {
                throw new JsonException("Failed to deserialize settings");
            }

            BotUsername = settings.BotUsername;
            AccessToken = settings.AccessToken;
            ChannelName = settings.ChannelName;
            AllowedUsers = settings.AllowedUsers;
            BlockedUsers = settings.BlockedUsers;
            MinSentenceWords = settings.MinSentenceWords;
            MaxSentenceWords = settings.MaxSentenceWords;
            AutoGenerateMessages = settings.AutoGenerateMessages;
            AutoGenerateInterval = settings.AutoGenerateInterval;
            AllowGenerateCommand = settings.AllowGenerateCommand;
            GenerateCommands = settings.GenerateCommands;
        }
        catch (JsonException e)
        {
            Console.WriteLine($"Failed to load settings: {e.Message}");
            Console.WriteLine("Generating default settings");
            GenerateDefaultSettings(path);
            return;
        }
    }

    /// <summary>
    /// Generates default settings and saves them to the specified path
    /// </summary>
    /// <param name="path">Path to save settings to</param>
    private void GenerateDefaultSettings(string path)
    {
        var settings = new Settings
        {
            BotUsername = "botUsername",
            AccessToken = "accessToken",
            ChannelName = "channelName",
            AllowedUsers = new List<string> { "allowedUser1", "allowedUser2" },
            BlockedUsers = new List<string> { "blockedUser1", "blockedUser2" },
            MinSentenceWords = -1,
            MaxSentenceWords = 20,
            AutoGenerateMessages = true,
            AutoGenerateInterval = 5000,
            AllowGenerateCommand = true,
            GenerateCommands = new List<string> { "!generate" }
        };

        var json = JsonSerializer.Serialize(settings);
        File.WriteAllText(path, json);
    }
}
