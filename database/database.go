package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, connStr string) (*Database, error) {
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	return &Database{pool: pool}, nil
}

func (d *Database) Close() {
	d.pool.Close()
}

// EnsureChannel upserts the channel and creates its table partitions if they
// don't exist yet. Must be called before any other operation on a channel.
func (d *Database) EnsureChannel(ctx context.Context, botUsername, channelName string) (int, error) {
	var channelID int
	err := d.pool.QueryRow(ctx, `
		INSERT INTO channels (channel_name, bot_username)
		VALUES ($1, $2)
		ON CONFLICT (channel_name) DO UPDATE SET bot_username = EXCLUDED.bot_username
		RETURNING id
	`, channelName, botUsername).Scan(&channelID)
	if err != nil {
		return 0, fmt.Errorf("upsert channel: %w", err)
	}

	for _, table := range []string{"message_chain", "markov_starts", "markov_grammar"} {
		_, err := d.pool.Exec(ctx, fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %s_%d PARTITION OF %s FOR VALUES IN (%d)`,
			table, channelID, table, channelID,
		))
		if err != nil {
			return 0, fmt.Errorf("create partition %s_%d: %w", table, channelID, err)
		}
	}

	return channelID, nil
}

func (d *Database) SaveMessageChainNode(ctx context.Context, channelID int, messageID, parentMessageID, messageText string, isBotMessage bool) error {
	if messageID == "" {
		return nil
	}

	var parent any
	if parentMessageID != "" {
		parent = parentMessageID
	}

	_, err := d.pool.Exec(ctx, `
		INSERT INTO message_chain (channel_id, message_id, parent_message_id, message_text, is_bot_message)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (channel_id, message_id) DO UPDATE SET
			parent_message_id = EXCLUDED.parent_message_id,
			message_text      = EXCLUDED.message_text,
			is_bot_message    = EXCLUDED.is_bot_message
	`, channelID, messageID, parent, messageText, isBotMessage)
	return err
}

func (d *Database) AddStart(ctx context.Context, channelID int, word1, word2 string) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO markov_starts (channel_id, word1, word2, count)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (channel_id, word1, word2) DO UPDATE SET count = markov_starts.count + 1
	`, channelID, word1, word2)
	return err
}

// AddGrammar trains a grammar entry. Pass nil for word3 to record a terminal
// state (end of sentence).
func (d *Database) AddGrammar(ctx context.Context, channelID int, word1, word2 string, word3 *string) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO markov_grammar (channel_id, word1, word2, word3, count)
		VALUES ($1, $2, $3, $4, 1)
		ON CONFLICT (channel_id, word1, word2, word3)
		DO UPDATE SET count = markov_grammar.count + 1
	`, channelID, word1, word2, word3)
	return err
}

// GetNextWord returns the next word after word1+word2. Returns "" when the
// chain terminates (NULL word3) or no transition exists — both mean stop.
func (d *Database) GetNextWord(ctx context.Context, channelID int, word1, word2 string) string {
	if word1 == "" || word2 == "" {
		return ""
	}

	var word3 *string
	err := d.pool.QueryRow(ctx, `
		SELECT word3 FROM markov_grammar
		WHERE channel_id = $1 AND word1 = $2 AND word2 = $3
		ORDER BY RANDOM()
		LIMIT 1
	`, channelID, word1, word2).Scan(&word3)
	if err != nil || word3 == nil {
		return ""
	}
	return *word3
}

func (d *Database) GetStartWord(ctx context.Context, channelID int) string {
	var word1, word2 string
	err := d.pool.QueryRow(ctx, `
		SELECT word1, word2 FROM markov_starts
		WHERE channel_id = $1
		ORDER BY RANDOM()
		LIMIT 1
	`, channelID).Scan(&word1, &word2)
	if err != nil {
		return ""
	}
	return word1 + " " + word2
}

func (d *Database) GetStatistics(ctx context.Context, channelID int) map[string]int {
	stats := make(map[string]int)

	var count int
	if err := d.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM markov_starts WHERE channel_id = $1`, channelID,
	).Scan(&count); err == nil {
		stats["TotalStartPairs"] = count
	}

	if err := d.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM markov_grammar WHERE channel_id = $1`, channelID,
	).Scan(&count); err == nil {
		stats["TotalGrammarEntries"] = count
	}

	return stats
}

// DeleteMessageChain removes the message and all its Twitch replies, undoing
// their Markov training. tokenize must be the same function used during training.
func (d *Database) DeleteMessageChain(ctx context.Context, channelID int, rootMessageID string, tokenize func(string) []string) error {
	if rootMessageID == "" {
		return nil
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	rows, err := tx.Query(ctx, `
		WITH RECURSIVE chain AS (
			SELECT message_id, message_text, is_bot_message
			FROM message_chain
			WHERE channel_id = $1 AND message_id = $2
			UNION ALL
			SELECT mc.message_id, mc.message_text, mc.is_bot_message
			FROM message_chain mc
			JOIN chain c ON mc.parent_message_id = c.message_id
			WHERE mc.channel_id = $1
		)
		SELECT message_id, message_text, is_bot_message FROM chain
	`, channelID, rootMessageID)
	if err != nil {
		return err
	}

	type chainMessage struct {
		id           string
		text         string
		isBotMessage bool
	}
	var messages []chainMessage
	for rows.Next() {
		var m chainMessage
		if err = rows.Scan(&m.id, &m.text, &m.isBotMessage); err != nil {
			rows.Close()
			return err
		}
		messages = append(messages, m)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}

	for _, m := range messages {
		if m.isBotMessage {
			continue
		}
		tokens := tokenize(m.text)
		if len(tokens) < 2 {
			continue
		}
		if err = decrementStart(ctx, tx, channelID, tokens[0], tokens[1]); err != nil {
			return err
		}
		for i := 0; i < len(tokens)-2; i++ {
			w3 := tokens[i+2]
			if err = decrementGrammar(ctx, tx, channelID, tokens[i], tokens[i+1], &w3); err != nil {
				return err
			}
		}
		if err = decrementGrammar(ctx, tx, channelID, tokens[len(tokens)-2], tokens[len(tokens)-1], nil); err != nil {
			return err
		}
	}

	if _, err = tx.Exec(ctx, `
		WITH RECURSIVE chain AS (
			SELECT message_id FROM message_chain
			WHERE channel_id = $1 AND message_id = $2
			UNION ALL
			SELECT mc.message_id
			FROM message_chain mc
			JOIN chain c ON mc.parent_message_id = c.message_id
			WHERE mc.channel_id = $1
		)
		DELETE FROM message_chain
		WHERE channel_id = $1 AND message_id IN (SELECT message_id FROM chain)
	`, channelID, rootMessageID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func decrementStart(ctx context.Context, tx pgx.Tx, channelID int, word1, word2 string) error {
	if _, err := tx.Exec(ctx,
		`UPDATE markov_starts SET count = count - 1 WHERE channel_id=$1 AND word1=$2 AND word2=$3`,
		channelID, word1, word2,
	); err != nil {
		return err
	}
	_, err := tx.Exec(ctx,
		`DELETE FROM markov_starts WHERE channel_id=$1 AND word1=$2 AND word2=$3 AND count <= 0`,
		channelID, word1, word2,
	)
	return err
}

func decrementGrammar(ctx context.Context, tx pgx.Tx, channelID int, word1, word2 string, word3 *string) error {
	if _, err := tx.Exec(ctx,
		`UPDATE markov_grammar SET count = count - 1
		 WHERE channel_id=$1 AND word1=$2 AND word2=$3 AND word3 IS NOT DISTINCT FROM $4`,
		channelID, word1, word2, word3,
	); err != nil {
		return err
	}
	_, err := tx.Exec(ctx,
		`DELETE FROM markov_grammar
		 WHERE channel_id=$1 AND word1=$2 AND word2=$3 AND word3 IS NOT DISTINCT FROM $4 AND count <= 0`,
		channelID, word1, word2, word3,
	)
	return err
}
