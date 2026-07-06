-- +goose Up
CREATE TABLE markov_grammar (
    channel_id INT  NOT NULL,
    word1      TEXT NOT NULL,
    word2      TEXT NOT NULL,
    word3      TEXT,
    count      INT  NOT NULL DEFAULT 1,
    UNIQUE NULLS NOT DISTINCT (channel_id, word1, word2, word3)
) PARTITION BY LIST (channel_id);

COMMENT ON TABLE markov_grammar IS 'Word trigrams forming the Markov chain transitions. Given word1 and word2, word3 is the next word to emit.';
COMMENT ON COLUMN markov_grammar.channel_id IS 'Channel this trigram belongs to.';
COMMENT ON COLUMN markov_grammar.word1 IS 'First word of the state bigram.';
COMMENT ON COLUMN markov_grammar.word2 IS 'Second word of the state bigram.';
COMMENT ON COLUMN markov_grammar.word3 IS 'Next word to emit. NULL marks the end of a sentence.';
COMMENT ON COLUMN markov_grammar.count IS 'Number of times this trigram has been observed.';

-- +goose Down
DROP TABLE markov_grammar;
