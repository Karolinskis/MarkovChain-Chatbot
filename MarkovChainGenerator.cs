using System;
using System.Collections.Generic;
using System.Linq;
using MarkovChainChatbot.Utils;

public class MarkovChainGenerator
{
    private Database _database;
    private List<string> _blacklistedWords;

    public MarkovChainGenerator(Database database, List<string> blacklistedWords)
    {
        _database = database;
        _blacklistedWords = blacklistedWords;
    }

    public void Train(List<string> tokens)
    {
        if (tokens.Count < 2) return;

        _database.AddStart(tokens[0], tokens[1]);

        for (int i = 0; i < tokens.Count - 2; i++)
        {
            _database.AddGrammar(tokens[i], tokens[i + 1], tokens[i + 2]);
        }

        // Add <END> to the grammar at the end of the sentence
        if (tokens.Count >= 2)
        {
            _database.AddGrammar(tokens[tokens.Count - 2], tokens[tokens.Count - 1], "<END>");
        }
    }

    public List<string> GenerateSentence(string startWord, int length)
    {
        for (int attempt = 0; attempt < 3; attempt++)
        {
            var words = startWord.Split(' ');
            if (words.Length < 2) return startWord.Split(' ').ToList();

            var currentWord1 = words[0];
            var currentWord2 = words[1];

            var result = new List<string> { currentWord1, currentWord2 };

            for (int i = 0; i < length - 2; i++)
            {
                var nextWord = _database.GetNextWord(currentWord1, currentWord2);
                if (string.IsNullOrEmpty(nextWord) || nextWord == "<END>") break;

                if (_blacklistedWords.Any(blacklistedWord => MessageParser.Normalize(nextWord).Contains(MessageParser.Normalize(blacklistedWord))))
                {
                    Logger.Instance.Log($"Sentence: {string.Join(" ", result)}. Blacklisted word: {nextWord}", sendToDiscord: false);
                    break;
                }

                result.Add(nextWord);
                currentWord1 = currentWord2;
                currentWord2 = nextWord;
            }

            if (!result.Any(words => _blacklistedWords.Contains(MessageParser.Normalize(words))))
            {
                return result;
            }
        }

        Logger.Instance.Log($"Failed to generate sentence after 3 attempts. Start word: {startWord}", sendToDiscord: true);

        return new List<string>();
    }

    public string GenerateMessage()
    {
        var startWord = _database.GetStartWord();
        if (startWord == null) return string.Empty;

        return Tokenizer.Detokenize(GenerateSentence(startWord, Settings.Instance.MaxSentenceWords));
    }
}
