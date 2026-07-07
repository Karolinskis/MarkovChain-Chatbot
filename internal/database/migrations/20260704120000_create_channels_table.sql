-- +goose Up
CREATE TABLE channels (
    id           SERIAL PRIMARY KEY,
    channel_name TEXT UNIQUE NOT NULL,
    bot_username TEXT NOT NULL
);

COMMENT ON TABLE channels IS 'Tracks every Twitch channel the bot is active in.';
COMMENT ON COLUMN channels.id IS 'Internal channel ID, used as the partition key in all other tables.';
COMMENT ON COLUMN channels.channel_name IS 'Twitch channel login name.';
COMMENT ON COLUMN channels.bot_username IS 'Which bot account serves this channel.';

-- +goose Down
DROP TABLE channels;
