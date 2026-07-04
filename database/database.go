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
	if _, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS MessageChain (
			message_id TEXT PRIMARY KEY,
			parent_message_id TEXT,
			is_bot_message INTEGER NOT NULL DEFAULT 0
		);`); err != nil {
		return err
	}

	if _, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS MessageStartContribution (
			message_id TEXT NOT NULL,
			word1 TEXT COLLATE NOCASE,
			word2 TEXT COLLATE NOCASE,
			count INTEGER NOT NULL,
			PRIMARY KEY (message_id, word1, word2)
		);`); err != nil {
		return err
	}

	if _, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS MessageGrammarContribution (
			message_id TEXT NOT NULL,
			word1 TEXT COLLATE NOCASE,
			word2 TEXT COLLATE NOCASE,
			word3 TEXT COLLATE NOCASE,
			count INTEGER NOT NULL,
			PRIMARY KEY (message_id, word1, word2, word3)
		);`); err != nil {
		return err
	}

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
	return d.addStartWithDelta(d.db, word1, word2, 1)
}

func (d *Database) addStartWithDelta(exec sqlExecer, word1, word2 string, delta int) error {
	tableName := fmt.Sprintf("MarkovStart%c", getSuffix(firstRune(word1)))
	query := fmt.Sprintf(`
		INSERT INTO %s (word1, word2, count)
		VALUES (?, ?, ?)
		ON CONFLICT(word1, word2) DO UPDATE SET count = count + excluded.count;`, tableName)
	_, err := exec.Exec(query, word1, word2, delta)
	return err
}

func (d *Database) AddGrammar(word1, word2, word3 string) error {
	return d.addGrammarWithDelta(d.db, word1, word2, word3, 1)
}

func (d *Database) addGrammarWithDelta(exec sqlExecer, word1, word2, word3 string, delta int) error {
	tableName := fmt.Sprintf("MarkovGrammar%c%c", getSuffix(firstRune(word1)), getSuffix(firstRune(word2)))
	query := fmt.Sprintf(`
		INSERT INTO %s (word1, word2, word3, count)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(word1, word2, word3) DO UPDATE SET count = count + excluded.count;`, tableName)
	_, err := exec.Exec(query, word1, word2, word3, delta)
	return err
}

func (d *Database) AddStartForMessage(messageID, word1, word2 string) error {
	if err := d.AddStart(word1, word2); err != nil {
		return err
	}

	if messageID == "" {
		return nil
	}

	_, err := d.db.Exec(`
		INSERT INTO MessageStartContribution (message_id, word1, word2, count)
		VALUES (?, ?, ?, 1)
		ON CONFLICT(message_id, word1, word2) DO UPDATE SET count = count + 1;`,
		messageID, word1, word2,
	)
	return err
}

func (d *Database) AddGrammarForMessage(messageID, word1, word2, word3 string) error {
	if err := d.AddGrammar(word1, word2, word3); err != nil {
		return err
	}

	if messageID == "" {
		return nil
	}

	_, err := d.db.Exec(`
		INSERT INTO MessageGrammarContribution (message_id, word1, word2, word3, count)
		VALUES (?, ?, ?, ?, 1)
		ON CONFLICT(message_id, word1, word2, word3) DO UPDATE SET count = count + 1;`,
		messageID, word1, word2, word3,
	)
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
	// Shuffle table order to avoid bias, then pick from the first non empty table
	order := rand.Perm(len(characters))
	for _, idx := range order {
		tableName := fmt.Sprintf("MarkovStart%c", characters[idx])
		query := fmt.Sprintf(`
			SELECT word1, word2 FROM %s
			ORDER BY RANDOM()
			LIMIT 1;`, tableName)

		var word1, word2 string
		err := d.db.QueryRow(query).Scan(&word1, &word2)
		if err == nil {
			return word1 + " " + word2
		}
	}
	return ""
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

func (d *Database) SaveMessageChainNode(messageID, parentMessageID string, isBotMessage bool) error {
	if messageID == "" {
		return nil
	}

	query := `
		INSERT INTO MessageChain (message_id, parent_message_id, is_bot_message)
		VALUES (?, ?, ?)
		ON CONFLICT(message_id) DO UPDATE SET
			parent_message_id = excluded.parent_message_id,
			is_bot_message = excluded.is_bot_message;`

	botFlag := 0
	if isBotMessage {
		botFlag = 1
	}

	_, err := d.db.Exec(query, messageID, nullableString(parentMessageID), botFlag)
	return err
}

func (d *Database) IsBotMessage(messageID string) (bool, error) {
	var botFlag int
	err := d.db.QueryRow(`SELECT is_bot_message FROM MessageChain WHERE message_id = ?;`, messageID).Scan(&botFlag)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return botFlag == 1, nil
}

func (d *Database) DeleteMessageChain(rootMessageID string) error {
	if rootMessageID == "" {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	startRows, err := tx.Query(`
		WITH RECURSIVE chain(message_id) AS (
			SELECT ?

			UNION ALL

			SELECT mc.message_id
			FROM MessageChain mc
			JOIN chain c ON mc.parent_message_id = c.message_id
		)
		SELECT sc.word1, sc.word2, SUM(sc.count)
		FROM MessageStartContribution sc
		JOIN chain c ON sc.message_id = c.message_id
		GROUP BY sc.word1, sc.word2;`, rootMessageID)
	if err != nil {
		return err
	}

	for startRows.Next() {
		var word1, word2 string
		var delta int
		if err = startRows.Scan(&word1, &word2, &delta); err != nil {
			startRows.Close()
			return err
		}
		if err = d.decrementStartCount(tx, word1, word2, delta); err != nil {
			startRows.Close()
			return err
		}
	}
	if err = startRows.Close(); err != nil {
		return err
	}

	grammarRows, err := tx.Query(`
		WITH RECURSIVE chain(message_id) AS (
			SELECT ?

			UNION ALL

			SELECT mc.message_id
			FROM MessageChain mc
			JOIN chain c ON mc.parent_message_id = c.message_id
		)
		SELECT gc.word1, gc.word2, gc.word3, SUM(gc.count)
		FROM MessageGrammarContribution gc
		JOIN chain c ON gc.message_id = c.message_id
		GROUP BY gc.word1, gc.word2, gc.word3;`, rootMessageID)
	if err != nil {
		return err
	}

	for grammarRows.Next() {
		var word1, word2, word3 string
		var delta int
		if err = grammarRows.Scan(&word1, &word2, &word3, &delta); err != nil {
			grammarRows.Close()
			return err
		}
		if err = d.decrementGrammarCount(tx, word1, word2, word3, delta); err != nil {
			grammarRows.Close()
			return err
		}
	}
	if err = grammarRows.Close(); err != nil {
		return err
	}

	if _, err = tx.Exec(`
		WITH RECURSIVE chain(message_id) AS (
			SELECT ?

			UNION ALL

			SELECT mc.message_id
			FROM MessageChain mc
			JOIN chain c ON mc.parent_message_id = c.message_id
		)
		DELETE FROM MessageStartContribution
		WHERE message_id IN (SELECT message_id FROM chain);`, rootMessageID); err != nil {
		return err
	}

	if _, err = tx.Exec(`
		WITH RECURSIVE chain(message_id) AS (
			SELECT ?

			UNION ALL

			SELECT mc.message_id
			FROM MessageChain mc
			JOIN chain c ON mc.parent_message_id = c.message_id
		)
		DELETE FROM MessageGrammarContribution
		WHERE message_id IN (SELECT message_id FROM chain);`, rootMessageID); err != nil {
		return err
	}

	if _, err = tx.Exec(`
		WITH RECURSIVE chain(message_id) AS (
			SELECT ?

			UNION ALL

			SELECT mc.message_id
			FROM MessageChain mc
			JOIN chain c ON mc.parent_message_id = c.message_id
		)
		DELETE FROM MessageChain
		WHERE message_id IN (SELECT message_id FROM chain);`, rootMessageID); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func (d *Database) decrementStartCount(exec sqlExecer, word1, word2 string, delta int) error {
	tableName := fmt.Sprintf("MarkovStart%c", getSuffix(firstRune(word1)))
	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE word1 = ? AND word2 = ? AND count <= ?;`, tableName)
	if _, err := exec.Exec(deleteQuery, word1, word2, delta); err != nil {
		return err
	}

	updateQuery := fmt.Sprintf(`UPDATE %s SET count = count - ? WHERE word1 = ? AND word2 = ? AND count > ?;`, tableName)
	_, err := exec.Exec(updateQuery, delta, word1, word2, delta)
	return err
}

func (d *Database) decrementGrammarCount(exec sqlExecer, word1, word2, word3 string, delta int) error {
	tableName := fmt.Sprintf("MarkovGrammar%c%c", getSuffix(firstRune(word1)), getSuffix(firstRune(word2)))
	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE word1 = ? AND word2 = ? AND word3 = ? AND count <= ?;`, tableName)
	if _, err := exec.Exec(deleteQuery, word1, word2, word3, delta); err != nil {
		return err
	}

	updateQuery := fmt.Sprintf(`UPDATE %s SET count = count - ? WHERE word1 = ? AND word2 = ? AND word3 = ? AND count > ?;`, tableName)
	_, err := exec.Exec(updateQuery, delta, word1, word2, word3, delta)
	return err
}

type sqlExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
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
