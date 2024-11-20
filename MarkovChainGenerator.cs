using System;
using System.Collections.Generic;
using System.Linq;
using MarkovChainChatbot.Utils;

public class MarkovChainGenerator
{
    private Database _database;

    public MarkovChainGenerator(Database database)
    {
        _database = database;
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
        var words = startWord.Split(' ');
        if (words.Length < 2) return startWord.Split(' ').ToList();

        var currentWord1 = words[0];
        var currentWord2 = words[1];

        var result = new List<string> { currentWord1, currentWord2 };

        for (int i = 0; i < length - 2; i++)
        {
            var nextWord = _database.GetNextWord(currentWord1, currentWord2);
            if (string.IsNullOrEmpty(nextWord) || nextWord == "<END>") break;

            result.Add(nextWord);
            currentWord1 = currentWord2;
            currentWord2 = nextWord;
        }

        return result;
    }

    public string GenerateMessage()
    {
        var startWord = _database.GetStartWord();
        if (startWord == null) return string.Empty;

        return Tokenizer.Detokenize(GenerateSentence(startWord, Settings.Instance.MaxSentenceWords));
    }
}
