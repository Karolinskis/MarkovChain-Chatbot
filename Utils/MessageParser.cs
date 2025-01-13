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
            message = Normalize(message);

            // Check for links
            if (Regex.IsMatch(message, @"http[^\s]+"))
            {
                Logger.Instance.Log($"Blocked message: {message}. Reason: contains link", sendToDiscord: false);
                return false;
            }

            // Check for mentions
            if (Regex.IsMatch(message, @"@\w+"))
            {
                Logger.Instance.Log($"Blocked message: {message}. Reason: contains mention", sendToDiscord: false);
                return false;
            }

            // Check for commands
            if (Regex.IsMatch(message, @"^[!.,]\w+"))
            {
                Logger.Instance.Log($"Blocked message: {message}. Reason: contains command", sendToDiscord: false);
                return false;
            }

            return true;
        }

        /// <summary>
        /// Normalizes a string by replacing diacritical marks (accents) with their base characters.
        /// </summary>
        /// <param name="input">String to normalize.</param>
        /// <returns>The normalized string without diacritical marks.</returns>
        public static string Normalize(string input)
        {
            StringBuilder normalized = new StringBuilder();
            foreach (char c in input)
            {
                if (char.IsSurrogate(c) || char.IsSymbol(c) || char.IsPunctuation(c))
                {
                    normalized.Append(c);
                }
                else
                {
                    normalized.Append(Normalize(c));
                }
            }

            return normalized.ToString();
        }

        /// <summary>
        /// Normalizes a character by removing any diacritical marks (accents) and returning the base character.
        /// </summary>
        /// <param name="character">The character to normalize.</param>
        /// <returns>The normalized character without diacritical marks, or the original character if no normalization is needed.</returns>
        public static char Normalize(char character)
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
