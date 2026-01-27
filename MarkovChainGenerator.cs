using System;
using System.Collections.Generic;
using System.Linq;
using MarkovChainChatbot.Utils;

public class MarkovChainGenerator
{
    private readonly Database _database;
    private readonly List<string> _normalizedBlacklistedWords;
    private readonly int _maxSentenceWords;
    private const int MaxGenerationAttempts = 10;

    public MarkovChainGenerator(Database database, List<string> blacklistedWords, int maxSentenceWords)
    {
        _database = database;
        _maxSentenceWords = maxSentenceWords;
        _normalizedBlacklistedWords = blacklistedWords?.Select(MessageParser.Normalize).ToList() ?? new List<string>();
    }

    public void Train(List<string> tokens)
    {
        if (tokens.Count < 2) return;

        _database.AddStart(tokens[0], tokens[1]);

        for (int i = 0; i < tokens.Count - 2; i++)
        {
            _database.AddGrammar(tokens[i], tokens[i + 1], tokens[i + 2]);
        }

        // <END> is used to signify the end of a sentence
        if (tokens.Count >= 2)
        {
            _database.AddGrammar(tokens[tokens.Count - 2], tokens[tokens.Count - 1], "<END>");
        }
    }

    public string GenerateMessage()
    {
        for (int i = 0; i < MaxGenerationAttempts; i++)
        {
            var startWordPair = _database.GetStartWord();
            if (string.IsNullOrEmpty(startWordPair)) continue;

            var sentence = TryGenerateSentence(startWordPair);
            if (sentence.Any())
            {
                return Tokenizer.Detokenize(sentence);
            }
        }

        Logger.Instance.Log($"Failed to generate a clean sentence after {MaxGenerationAttempts} attempts.", sendToDiscord: true);
        return string.Empty;
    }

    public Dictionary<string, int> GetStatistics()
    {
        return _database.GetStatistics();
    }

    private List<string> TryGenerateSentence(string startWordPair)
    {
        var words = startWordPair.Split(' ');
        if (words.Length < 2 || AreWordsBlacklisted(words))
        {
            return new List<string>();
        }

        var result = new List<string>(words);
        var currentWord1 = words[0];
        var currentWord2 = words[1];

        for (int i = 0; i < _maxSentenceWords - 2; i++)
        {
            var nextWord = _database.GetNextWord(currentWord1, currentWord2);
            if (string.IsNullOrEmpty(nextWord) || nextWord == "<END>") break;

            if (IsWordBlacklisted(nextWord))
            {
                Logger.Instance.Log($"Message: {string.Join(" ", result)}. Blacklisted word: {nextWord}", sendToDiscord: false);
                return new List<string>();
            }

            result.Add(nextWord);
            currentWord1 = currentWord2;
            currentWord2 = nextWord;
        }

        return result;
    }

    private bool IsWordBlacklisted(string word)
    {
        if (_normalizedBlacklistedWords.Count == 0) return false;
        var normalizedWord = MessageParser.Normalize(word);
        return _normalizedBlacklistedWords.Contains(normalizedWord);
    }

    private bool AreWordsBlacklisted(string[] words)
    {
        return words.Any(IsWordBlacklisted);
    }
}
