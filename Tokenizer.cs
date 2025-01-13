using System.Text.RegularExpressions;

public class Tokenizer
{
    private static readonly Regex EmoticonRegex = new Regex(
        @"(
            [<>]?
            [:;=8]                     # eyes
            [\-o\*\']?                 # optional nose
            [\)\]\(\[dDpP/\:\}\{@\|\\] # mouth
            |
            [\)\]\(\[dDpP/\:\}\{@\|\\] # mouth
            [\-o\*\']?                 # optional nose
            [:;=8]                     # eyes
            [<>]?
            |
            <3                         # heart
        )", RegexOptions.IgnorePatternWhitespace | RegexOptions.IgnoreCase | RegexOptions.Compiled);

    private static readonly Regex[] StartingQuotes = new[]
    {
        new Regex(@"([«“‘„]|[`]+)", RegexOptions.Compiled),
        new Regex(@"(``)", RegexOptions.Compiled),
        new Regex(@"(?i)(\')(?!re|ve|ll|m|t|s|d)(\w)\b", RegexOptions.Compiled)
    };

    private static readonly Regex[] Punctuation = new[]
    {
        new Regex(@"’", RegexOptions.Compiled),
        new Regex(@"([^\.])(\.)([\]\)}>" + "\u00BB\u201D\u2019" + @"]*)\s*$", RegexOptions.Compiled),
        new Regex(@"([:,])([^\d])", RegexOptions.Compiled),
        new Regex(@"([:,])$", RegexOptions.Compiled),
        new Regex(@"\.{2,}", RegexOptions.Compiled),
        new Regex(@"[;#$%&]", RegexOptions.Compiled),
        new Regex(@"([^\.])(\.)([\]\)}>" + "\u0022\u0027" + @"]*)\s*$", RegexOptions.Compiled),
        new Regex(@"[?!]", RegexOptions.Compiled),
        new Regex(@"([^'])' ", RegexOptions.Compiled),
        new Regex(@"[*]", RegexOptions.Compiled)
    };

    public static List<string> Tokenize(string sentence)
    {
        var output = new List<string>();
        var match = EmoticonRegex.Match(sentence);

        while (match.Success)
        {
            output.AddRange(TokenizePart(sentence.Substring(0, match.Index).Trim()));
            output.Add(match.Value);
            sentence = sentence.Substring(match.Index + match.Length).Trim();
            match = EmoticonRegex.Match(sentence);
        }

        output.AddRange(TokenizePart(sentence));
        return output;
    }

    private static List<string> TokenizePart(string sentence)
    {
        foreach (var regex in StartingQuotes)
        {
            sentence = regex.Replace(sentence, " $1 ");
        }

        foreach (var regex in Punctuation)
        {
            sentence = regex.Replace(sentence, " $0 ");
        }

        return new List<string>(sentence.Split(new[] { ' ' }, StringSplitOptions.RemoveEmptyEntries));
    }

    public static string Detokenize(List<string> tokenized)
    {
        var result = new List<string>();
        for (int i = 0; i < tokenized.Count; i++)
        {
            if (i > 0 && IsPunctuation(tokenized[i]))
            {
                result[result.Count - 1] += tokenized[i];
            }
            else
            {
                result.Add(tokenized[i]);
            }
        }

        return string.Join(" ", result);
    }

    private static bool IsPunctuation(string token)
    {
        return Punctuation.Any(regex => regex.IsMatch(token)) && !EmoticonRegex.IsMatch(token);
    }
}
