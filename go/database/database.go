package database

import (
	"database/sql"
	"fmt"
	"math/rand"
	"unicode"

	"markovchain-chatbot/filter"

	_ "modernc.org/sqlite"
)

var characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ_"

type Database struct {
	db *sql.DB
}

func New(databasePath string) (*Database, error) {
	db, err := sql.Open("sqlite", databasePath+"?_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	d := &Database{db: db}
	if err := d.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return d, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) initialize() error {
	for _, firstChar := range characters {
		tableName := fmt.Sprintf("MarkovStart%c", firstChar)
		query := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				word1 TEXT COLLATE NOCASE,
				word2 TEXT COLLATE NOCASE,
				count INTEGER,
				PRIMARY KEY (word1, word2)
			);`, tableName)
		if _, err := d.db.Exec(query); err != nil {
			return err
		}

		for _, secondChar := range characters {
			tableName := fmt.Sprintf("MarkovGrammar%c%c", firstChar, secondChar)
			query := fmt.Sprintf(`
				CREATE TABLE IF NOT EXISTS %s (
					word1 TEXT COLLATE NOCASE,
					word2 TEXT COLLATE NOCASE,
					word3 TEXT COLLATE NOCASE,
					count INTEGER,
					PRIMARY KEY (word1, word2, word3)
				);`, tableName)
			if _, err := d.db.Exec(query); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Database) AddStart(word1, word2 string) error {
	tableName := fmt.Sprintf("MarkovStart%c", getSuffix(firstRune(word1)))
	query := fmt.Sprintf(`
		INSERT INTO %s (word1, word2, count)
		VALUES (?, ?, 1)
		ON CONFLICT(word1, word2) DO UPDATE SET count = count + 1;`, tableName)
	_, err := d.db.Exec(query, word1, word2)
	return err
}

func (d *Database) AddGrammar(word1, word2, word3 string) error {
	tableName := fmt.Sprintf("MarkovGrammar%c%c", getSuffix(firstRune(word1)), getSuffix(firstRune(word2)))
	query := fmt.Sprintf(`
		INSERT INTO %s (word1, word2, word3, count)
		VALUES (?, ?, ?, 1)
		ON CONFLICT(word1, word2, word3) DO UPDATE SET count = count + 1;`, tableName)
	_, err := d.db.Exec(query, word1, word2, word3)
	return err
}

func (d *Database) GetNextWord(word1, word2 string) string {
	if word1 == "" || word2 == "" {
		return ""
	}

	tableName := fmt.Sprintf("MarkovGrammar%c%c", getSuffix(firstRune(word1)), getSuffix(firstRune(word2)))
	query := fmt.Sprintf(`
		SELECT word3 FROM %s
		WHERE word1 = ? AND word2 = ?
		ORDER BY RANDOM()
		LIMIT 1;`, tableName)

	var result string
	err := d.db.QueryRow(query, word1, word2).Scan(&result)
	if err != nil {
		return ""
	}
	return result
}

func (d *Database) GetStartWord() string {
	idx := rand.Intn(len(characters))
	tableName := fmt.Sprintf("MarkovStart%c", characters[idx])
	query := fmt.Sprintf(`
		SELECT word1, word2 FROM %s
		ORDER BY RANDOM()
		LIMIT 1;`, tableName)

	var word1, word2 string
	err := d.db.QueryRow(query).Scan(&word1, &word2)
	if err != nil {
		return ""
	}
	return word1 + " " + word2
}

func (d *Database) GetStatistics() map[string]int {
	stats := make(map[string]int)

	totalStartPairs := 0
	for _, firstChar := range characters {
		tableName := fmt.Sprintf("MarkovStart%c", firstChar)
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s;", tableName)
		var count int
		if err := d.db.QueryRow(query).Scan(&count); err == nil {
			totalStartPairs += count
		}
	}
	stats["TotalStartPairs"] = totalStartPairs

	totalGrammarEntries := 0
	for _, firstChar := range characters {
		for _, secondChar := range characters {
			tableName := fmt.Sprintf("MarkovGrammar%c%c", firstChar, secondChar)
			query := fmt.Sprintf("SELECT COUNT(*) FROM %s;", tableName)
			var count int
			if err := d.db.QueryRow(query).Scan(&count); err == nil {
				totalGrammarEntries += count
			}
		}
	}
	stats["TotalGrammarEntries"] = totalGrammarEntries

	return stats
}

func getSuffix(character rune) byte {
	if !unicode.IsLetter(character) {
		return '_'
	}
	normalized := unicode.ToUpper(filter.NormalizeChar(character))
	if normalized >= 'A' && normalized <= 'Z' {
		return byte(normalized)
	}
	return '_'
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return '_'
}
