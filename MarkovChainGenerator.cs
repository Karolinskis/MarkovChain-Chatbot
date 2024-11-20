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

    public void Train(string input)
    {
        input = MessageParser.CleanMessage(input);

        var words = input.Split(' ');
        if (words.Length < 2) return;

        _database.AddStart(words[0], words[1]);

        for (int i = 0; i < words.Length - 2; i++)
        {
            _database.AddGrammar(words[i], words[i + 1], words[i + 2]);
        }
    }

    public string GenerateSentence(string startWord, int length)
    {
        var words = startWord.Split(' ');
        if (words.Length < 2) return startWord;

        var currentWord1 = words[0];
        var currentWord2 = words[1];

        var result = new List<string> { currentWord1, currentWord2 };

        for (int i = 0; i < length - 2; i++)
        {
            var nextWord = _database.GetNextWord(currentWord1, currentWord2);
            if (nextWord == null) break;

            result.Add(nextWord);
            currentWord1 = currentWord2;
            currentWord2 = nextWord;
        }

        return string.Join(" ", result);
    }

    public string GenerateMessage()
    {
        var startWord = _database.GetStartWord();
        if (startWord == null) return string.Empty;

        return GenerateSentence(startWord, 10);
    }
}
