using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;

public class Settings
{
    private static Settings _instance;
    private static readonly object _lock = new object();

    public string BotUsername { get; set; }
    public string AccessToken { get; set; }
    public string ChannelName { get; set; }
    public bool TrainingMode { get; set; }
    public List<string>? AllowedUsers { get; set; }
    public List<string>? BlockedUsers { get; set; }
    public int MinSentenceWords { get; set; }
    public int MaxSentenceWords { get; set; }
    public bool AutoGenerateMessages { get; set; }
    public int AutoGenerateInterval { get; set; }
    public bool AllowGenerateCommand { get; set; }
    public List<string> GenerateCommands { get; set; }
    public List<string> BlacklistedWords { get; set; }
    public bool EnableDiscordLogging { get; set; }
    public string? DiscordWebhookUrl { get; set; }
    public bool AllowNonAsciiMessages { get; set; }

    public Settings() { }

    public static Settings Instance
    {
        get
        {
            lock (_lock)
            {
                if (_instance == null)
                {
                    _instance = new Settings();
                }

                return _instance;
            }
        }
    }

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
            TrainingMode = settings.TrainingMode;
            AllowedUsers = settings.AllowedUsers;
            BlockedUsers = settings.BlockedUsers;
            MinSentenceWords = settings.MinSentenceWords;
            MaxSentenceWords = settings.MaxSentenceWords;
            AutoGenerateMessages = settings.AutoGenerateMessages;
            AutoGenerateInterval = settings.AutoGenerateInterval;
            AllowGenerateCommand = settings.AllowGenerateCommand;
            GenerateCommands = settings.GenerateCommands;
            BlacklistedWords = settings.BlacklistedWords;
            EnableDiscordLogging = settings.EnableDiscordLogging;
            DiscordWebhookUrl = settings.DiscordWebhookUrl;
            AllowNonAsciiMessages = settings.AllowNonAsciiMessages;
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
            TrainingMode = false,
            AllowedUsers = new List<string> { "allowedUser1", "allowedUser2" },
            BlockedUsers = new List<string> { "blockedUser1", "blockedUser2" },
            MinSentenceWords = -1,
            MaxSentenceWords = 20,
            AutoGenerateMessages = true,
            AutoGenerateInterval = 5000,
            AllowGenerateCommand = true,
            GenerateCommands = new List<string> { "!generate" },
            BlacklistedWords = new List<string> { },
            EnableDiscordLogging = false,
            DiscordWebhookUrl = "discordWebhookUrl",
            AllowNonAsciiMessages = false
        };

        var json = JsonSerializer.Serialize(settings);
        File.WriteAllText(path, json);
    }
}
