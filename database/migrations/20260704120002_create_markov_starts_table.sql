-- +goose Up
CREATE TABLE markov_starts (
    channel_id INT  NOT NULL,
    word1      TEXT NOT NULL,
    word2      TEXT NOT NULL,
    count      INT  NOT NULL DEFAULT 1,
    PRIMARY KEY (channel_id, word1, word2)
) PARTITION BY LIST (channel_id);

COMMENT ON TABLE markov_starts IS 'Word bigrams that are valid sentence starters, weighted by how often they appear at the start of a message.';
COMMENT ON COLUMN markov_starts.channel_id IS 'Channel this bigram belongs to.';
COMMENT ON COLUMN markov_starts.word1 IS 'First word of the starting bigram.';
COMMENT ON COLUMN markov_starts.word2 IS 'Second word of the starting bigram.';
COMMENT ON COLUMN markov_starts.count IS 'Number of times this bigram has started a sentence.';

-- +goose Down
DROP TABLE markov_starts;
