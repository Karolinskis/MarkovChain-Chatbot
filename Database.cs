using System;
using System.Data.SQLite;
using System.IO;
using MarkovChainChatbot.Utils;

public class Database
{
    private List<char> _characters = new List<char>("ABCDEFGHIJKLMNOPQRSTUVWXYZ_");

    private string _connectionString;

    public Database(string databasePath)
    {
        //_databasePath = databasePath;
        _connectionString = $"Data Source={databasePath};Version=3;";
        InitializeDatabase(databasePath);
    }

    private void InitializeDatabase(string databasePath)
    {
        if (!File.Exists(databasePath))
        {
            SQLiteConnection.CreateFile(databasePath);
        }

        using (var connection = new SQLiteConnection(_connectionString))
        {
            connection.Open();

            foreach (var firstChar in _characters)
            {
                string createMarkovStartTable = $@"
                CREATE TABLE IF NOT EXISTS MarkovStart{firstChar} (
                    word1 TEXT COLLATE NOCASE,
                    word2 TEXT COLLATE NOCASE,
                    count INTEGER,
                    PRIMARY KEY (word1, word2)
                );";
                ExecuteNonQuery(connection, createMarkovStartTable);

                foreach (var secondChar in _characters)
                {
                    string createMarkovGrammarTable = $@"
                CREATE TABLE IF NOT EXISTS MarkovGrammar{firstChar}{secondChar} (
                    word1 TEXT COLLATE NOCASE,
                    word2 TEXT COLLATE NOCASE,
                    word3 TEXT COLLATE NOCASE,
                    count INTEGER,
                    PRIMARY KEY (word1, word2, word3)
                );";
                    ExecuteNonQuery(connection, createMarkovGrammarTable);
                }
            }
        }
    }

    public void AddStart(string word1, string word2)
    {
        using (var connection = new SQLiteConnection(_connectionString))
        {
            connection.Open();

            string tableName = $"MarkovStart{GetSuffix(word1[0])}";
            string insertStart = $@"
                INSERT INTO {tableName} (word1, word2, count)
                VALUES (@word1, @word2, 1)
                ON CONFLICT(word1, word2) DO UPDATE SET count = count + 1;";

            using (var command = new SQLiteCommand(insertStart, connection))
            {
                command.Parameters.AddWithValue("@word1", word1);
                command.Parameters.AddWithValue("@word2", word2);
                command.ExecuteNonQuery();
            }
        }
    }

    public void AddGrammar(string word1, string word2, string word3)
    {
        using (var connection = new SQLiteConnection(_connectionString))
        {
            connection.Open();

            string tableName = $"MarkovGrammar{GetSuffix(word1[0])}{GetSuffix(word2[0])}";
            string query = $@"
            INSERT INTO {tableName} (word1, word2, word3, count)
            VALUES (@word1, @word2, @word3, 1)
            ON CONFLICT(word1, word2, word3) DO UPDATE SET count = count + 1;";

            using (var command = new SQLiteCommand(query, connection))
            {
                command.Parameters.AddWithValue("@word1", word1);
                command.Parameters.AddWithValue("@word2", word2);
                command.Parameters.AddWithValue("@word3", word3);
                command.ExecuteNonQuery();
            }
        }
    }

    public string GetNextWord(string word1, string word2)
    {
        if (string.IsNullOrEmpty(word1) || string.IsNullOrEmpty(word2))
        {
            return string.Empty;
        }

        using (var connection = new SQLiteConnection(_connectionString))
        {
            connection.Open();

            string tableName = $"MarkovGrammar{GetSuffix(word1[0])}{GetSuffix(word2[0])}";
            string query = $@"
            SELECT word3
            FROM {tableName}
            WHERE word1 = @word1 AND word2 = @word2
            ORDER BY RANDOM()
            LIMIT 1;";

            using (var command = new SQLiteCommand(query, connection))
            {
                command.Parameters.AddWithValue("@word1", word1);
                command.Parameters.AddWithValue("@word2", word2);
                var result = command.ExecuteScalar();
                return result?.ToString() ?? string.Empty;
            }
        }
    }

    public string GetStartWord()
    {
        using (var connection = new SQLiteConnection(_connectionString))
        {
            connection.Open();

            string tableName = $"MarkovStart{GetRandomCharacter()}";
            string query = $@"
            SELECT word1, word2 FROM {tableName}
            ORDER BY RANDOM()
            LIMIT 1;";

            using (var command = new SQLiteCommand(query, connection))
            {
                using (var reader = command.ExecuteReader())
                {
                    if (reader.Read())
                    {
                        return reader["word1"].ToString() + " " + reader["word2"].ToString();
                    }
                }
            }
        }
        return string.Empty;
    }

    private char GetSuffix(char character)
    {
        if (!char.IsLetter(character))
        {
            return '_';
        }

        return MessageParser.Normalize(character);
    }

    private char GetRandomCharacter()
    {
        var random = new Random();
        return _characters[random.Next(_characters.Count)];
    }

    private void ExecuteNonQuery(SQLiteConnection connection, string query)
    {
        using (var command = new SQLiteCommand(query, connection))
        {
            command.ExecuteNonQuery();
        }
    }

}
