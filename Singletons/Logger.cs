using System.Text;
using System.Text.Json;

public class Logger
{
    private static Logger? _instance;
    private static readonly object _lock = new object();
    private readonly bool _enableDiscordLogging;
    private readonly string? _discordWebhookUrl;

    private Logger(bool enableDiscordLogging, string? discordWebhookUrl)
    {
        _enableDiscordLogging = enableDiscordLogging;
        _discordWebhookUrl = discordWebhookUrl;
    }

    public static Logger Instance
    {
        get
        {
            lock (_lock)
            {
                if (_instance == null)
                {
                    var settings = Settings.Instance;
                    _instance = new Logger(settings.EnableDiscordLogging, settings.DiscordWebhookUrl);
                }
                return _instance;
            }
        }
    }

    public void Log(string message, bool sendToDiscord = true)
    {
        string formattedMessage = $"[{DateTime.Now:yyyy-MM-dd HH:mm:ss}] {message}";
        Console.WriteLine(formattedMessage);

        if (_enableDiscordLogging)
        {
            if (sendToDiscord && !string.IsNullOrEmpty(_discordWebhookUrl))
            {
                SendDiscordMessage(message).Wait();
            }
        }
    }

    private async Task SendDiscordMessage(string message)
    {
        if (string.IsNullOrEmpty(_discordWebhookUrl))
        {
            return;
        }

        using (var httpClient = new HttpClient())
        {
            var payload = new
            {
                content = message
            };

            var content = new StringContent(JsonSerializer.Serialize(payload), Encoding.UTF8, "application/json");
            await httpClient.PostAsync(_discordWebhookUrl, content);
        }
    }
}
