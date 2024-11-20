using System.Text.RegularExpressions;

namespace MarkovChainChatbot.Utils
{
    public class MessageParser
    {
        /// <summary>
        /// Cleans the input message by removing links, mentions, and commands, and trimming whitespace.
        /// </summary>
        /// <param name="message">The input message to be cleaned.</param>
        /// <returns>The cleaned message.</returns>
        public static string CleanMessage(string message)
        {
            // Remove links
            message = Regex.Replace(message, @"http[^\s]+", string.Empty);

            // Remove mentions
            message = Regex.Replace(message, @"@\w+", string.Empty);

            // Remove commands
            message = Regex.Replace(message, @"^[!.,]\w+", string.Empty);

            // Trim whitespace
            message = message.Trim();

            return message;
        }
    }
}
