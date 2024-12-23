using System.Globalization;
using System.Text;
using System.Text.RegularExpressions;

namespace MarkovChainChatbot.Utils
{
    public class MessageParser
    {
        /// <summary>
        /// Checks if the input message is clean by verifying the absence of links, mentions, and commands.
        /// </summary>
        /// <param name="message">The input message to be checked.</param>
        /// <returns>True if the message is clean, otherwise false.</returns>
        public static bool IsCleanMessage(string message)
        {
            // Check for links
            if (Regex.IsMatch(message, @"http[^\s]+"))
            {
                Logger.Instance.Log($"Blocked message: {message}. Reason: contains link");
                return false;
            }

            // Check for mentions
            if (Regex.IsMatch(message, @"@\w+"))
            {
                Logger.Instance.Log($"Blocked message: {message}. Reason: contains mention");
                return false;
            }

            // Check for commands
            if (Regex.IsMatch(message, @"^[!.,]\w+"))
            {
                Logger.Instance.Log($"Blocked message: {message}. Reason: contains command");
                return false;
            }

            // Trim whitespace
            message = message.Trim();

            return true;
        }

        /// <summary>
        /// Normalizes a character by removing any diacritical marks (accents) and returning the base character.
        /// </summary>
        /// <param name="character">The character to normalize.</param>
        /// <returns>The normalized character without diacritical marks, or the original character if no normalization is needed.</returns>
        public static char NormalizeCharacter(char character)
        {
            string normalized = character.ToString().Normalize(NormalizationForm.FormD);
            foreach (char c in normalized)
            {
                if (CharUnicodeInfo.GetUnicodeCategory(c) != UnicodeCategory.NonSpacingMark)
                {
                    return c;
                }
            }

            return character;
        }
    }
}
