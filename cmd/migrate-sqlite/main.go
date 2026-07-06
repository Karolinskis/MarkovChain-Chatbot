package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "modernc.org/sqlite"
)

const (
	characters  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ_"
	endSentinel = "<END>"
)

func main() {
	var (
		sqlitePath  = flag.String("sqlite", "", "path to the SQLite database file")
		postgresURL = flag.String("postgres", "", "PostgreSQL connection string")
		channelName = flag.String("channel", "", "Twitch channel name (e.g. thexanos)")
		botUsername = flag.String("bot", "", "bot account username")
	)
	flag.Parse()

	if *sqlitePath == "" || *postgresURL == "" || *channelName == "" || *botUsername == "" {
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	sqlite, err := sql.Open("sqlite", *sqlitePath)
	if err != nil {
		slog.Error("open sqlite", "error", err)
		os.Exit(1)
	}
	defer sqlite.Close()

	pool, err := pgxpool.New(ctx, *postgresURL)
	if err != nil {
		slog.Error("open postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	channelID, err := ensureChannel(ctx, pool, *channelName, *botUsername)
	if err != nil {
		slog.Error("ensure channel", "error", err)
		os.Exit(1)
	}
	slog.Info("channel ready", "channel", *channelName, "channelID", channelID)

	if err := migrateStarts(ctx, sqlite, pool, channelID); err != nil {
		slog.Error("migrate starts", "error", err)
		os.Exit(1)
	}

	if err := migrateGrammar(ctx, sqlite, pool, channelID); err != nil {
		slog.Error("migrate grammar", "error", err)
		os.Exit(1)
	}

	if err := migrateMessageChain(ctx, sqlite, pool, channelID); err != nil {
		slog.Error("migrate message chain", "error", err)
		os.Exit(1)
	}

	slog.Info("migration complete")
}

func ensureChannel(ctx context.Context, pool *pgxpool.Pool, channelName, botUsername string) (int, error) {
	var channelID int
	err := pool.QueryRow(ctx, `
		INSERT INTO channels (channel_name, bot_username)
		VALUES ($1, $2)
		ON CONFLICT (channel_name) DO UPDATE SET bot_username = EXCLUDED.bot_username
		RETURNING id
	`, channelName, botUsername).Scan(&channelID)
	if err != nil {
		return 0, err
	}

	for _, table := range []string{"message_chain", "markov_starts", "markov_grammar"} {
		_, err := pool.Exec(ctx, fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %s_%d PARTITION OF %s FOR VALUES IN (%d)`,
			table, channelID, table, channelID,
		))
		if err != nil {
			return 0, fmt.Errorf("create partition %s_%d: %w", table, channelID, err)
		}
	}

	return channelID, nil
}

func migrateStarts(ctx context.Context, sqlite *sql.DB, pool *pgxpool.Pool, channelID int) error {
	type key struct{ word1, word2 string }
	counts := make(map[key]int)

	for _, ch := range characters {
		table := fmt.Sprintf("MarkovStart%c", ch)
		rows, err := sqlite.QueryContext(ctx, fmt.Sprintf("SELECT word1, word2, count FROM %s", table))
		if err != nil {
			return fmt.Errorf("query %s: %w", table, err)
		}
		for rows.Next() {
			var k key
			var count int
			if err := rows.Scan(&k.word1, &k.word2, &count); err != nil {
				rows.Close()
				return err
			}
			counts[k] += count
		}
		if err := rows.Close(); err != nil {
			return err
		}
	}

	if len(counts) == 0 {
		slog.Info("no starts to migrate")
		return nil
	}

	type row struct {
		word1, word2 string
		count        int
	}
	batch := make([]row, 0, len(counts))
	for k, count := range counts {
		batch = append(batch, row{k.word1, k.word2, count})
	}

	_, err := pool.CopyFrom(ctx,
		pgx.Identifier{"markov_starts"},
		[]string{"channel_id", "word1", "word2", "count"},
		pgx.CopyFromSlice(len(batch), func(i int) ([]any, error) {
			r := batch[i]
			return []any{channelID, r.word1, r.word2, r.count}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("copy starts: %w", err)
	}

	slog.Info("migrated starts", "count", len(batch))
	return nil
}

func migrateGrammar(ctx context.Context, sqlite *sql.DB, pool *pgxpool.Pool, channelID int) error {
	// key uses a sentinel for nil word3 since map keys can't be pointers
	type key struct{ word1, word2, word3 string }
	const nilWord3 = "\x00"
	counts := make(map[key]int)

	for _, c1 := range characters {
		for _, c2 := range characters {
			table := fmt.Sprintf("MarkovGrammar%c%c", c1, c2)
			rows, err := sqlite.QueryContext(ctx, fmt.Sprintf("SELECT word1, word2, word3, count FROM %s", table))
			if err != nil {
				return fmt.Errorf("query %s: %w", table, err)
			}
			for rows.Next() {
				var word1, word2, word3 string
				var count int
				if err := rows.Scan(&word1, &word2, &word3, &count); err != nil {
					rows.Close()
					return err
				}
				if word3 == endSentinel {
					word3 = nilWord3
				}
				counts[key{word1, word2, word3}] += count
			}
			if err := rows.Close(); err != nil {
				return err
			}
		}
	}

	if len(counts) == 0 {
		slog.Info("no grammar to migrate")
		return nil
	}

	type row struct {
		word1, word2 string
		word3        *string
		count        int
	}
	batch := make([]row, 0, len(counts))
	for k, count := range counts {
		r := row{word1: k.word1, word2: k.word2, count: count}
		if k.word3 != nilWord3 {
			w := k.word3
			r.word3 = &w
		}
		batch = append(batch, r)
	}

	_, err := pool.CopyFrom(ctx,
		pgx.Identifier{"markov_grammar"},
		[]string{"channel_id", "word1", "word2", "word3", "count"},
		pgx.CopyFromSlice(len(batch), func(i int) ([]any, error) {
			r := batch[i]
			return []any{channelID, r.word1, r.word2, r.word3, r.count}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("copy grammar: %w", err)
	}

	slog.Info("migrated grammar", "count", len(batch))
	return nil
}

func migrateMessageChain(ctx context.Context, sqlite *sql.DB, pool *pgxpool.Pool, channelID int) error {
	type row struct {
		messageID       string
		parentMessageID *string
		isBotMessage    bool
	}

	rows, err := sqlite.QueryContext(ctx, "SELECT message_id, parent_message_id, is_bot_message FROM MessageChain")
	if err != nil {
		return fmt.Errorf("query MessageChain: %w", err)
	}
	defer rows.Close()

	var batch []row
	for rows.Next() {
		var r row
		var botFlag int
		if err := rows.Scan(&r.messageID, &r.parentMessageID, &botFlag); err != nil {
			return err
		}
		r.isBotMessage = botFlag != 0
		batch = append(batch, r)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	if len(batch) == 0 {
		slog.Info("no message chain to migrate")
		return nil
	}

	// message_text is stored empty: the old schema never persisted raw message text,
	// so deletion untraining won't apply to these rows (tokenize("") returns < 2 tokens).
	_, err = pool.CopyFrom(ctx,
		pgx.Identifier{"message_chain"},
		[]string{"channel_id", "message_id", "parent_message_id", "message_text", "is_bot_message"},
		pgx.CopyFromSlice(len(batch), func(i int) ([]any, error) {
			r := batch[i]
			return []any{channelID, r.messageID, r.parentMessageID, "", r.isBotMessage}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("copy message chain: %w", err)
	}

	slog.Info("migrated message chain", "count", len(batch))
	return nil
}
