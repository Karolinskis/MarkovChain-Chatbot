using System;
using System.Data.SQLite;
using System.IO;

public class Database
{
    //private string _databasePath;
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

            string createMarkovStartTable = @"
                CREATE TABLE IF NOT EXISTS MarkovStart (
                    word1 TEXT COLLATE NOCASE,
                    word2 TEXT COLLATE NOCASE,
                    count INTEGER,
                    PRIMARY KEY (word1, word2)
                );";

            string createMarkovGrammarTable = @"
                CREATE TABLE IF NOT EXISTS MarkovGrammar (
                    word1 TEXT COLLATE NOCASE,
                    word2 TEXT COLLATE NOCASE,
                    word3 TEXT COLLATE NOCASE,
                    count INTEGER,
                    PRIMARY KEY (word1, word2, word3)
                );";

            using (var command = new SQLiteCommand(createMarkovStartTable, connection))
            {
                command.ExecuteNonQuery();
            }

            using (var command = new SQLiteCommand(createMarkovGrammarTable, connection))
            {
                command.ExecuteNonQuery();
            }
        }
    }

    public void AddStart(string word1, string word2)
    {
        using (var connection = new SQLiteConnection(_connectionString))
        {
            connection.Open();

            string insertStart = @"
                INSERT INTO MarkovStart (word1, word2, count)
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
            string query = @"
                INSERT INTO MarkovGrammar (word1, word2, word3, count)
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
        using (var connection = new SQLiteConnection(_connectionString))
        {
            connection.Open();

            string query = @"
                SELECT word3
                FROM MarkovGrammar
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
            string query = @"
                SELECT word1, word2 FROM MarkovStart
                ORDER BY RANDOM() LIMIT 1;";

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
}
