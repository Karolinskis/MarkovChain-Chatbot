-- +goose Up
CREATE TABLE message_chain (
    channel_id        INT     NOT NULL,
    message_id        TEXT    NOT NULL,
    parent_message_id TEXT,
    message_text      TEXT    NOT NULL,
    is_bot_message    BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (channel_id, message_id)
) PARTITION BY LIST (channel_id);

COMMENT ON TABLE message_chain IS 'Stores every Twitch message the bot sees. Used to walk reply chains when a message is deleted and undo the corresponding Markov training.';
COMMENT ON COLUMN message_chain.channel_id IS 'Channel this message belongs to.';
COMMENT ON COLUMN message_chain.message_id IS 'Twitch message UUID.';
COMMENT ON COLUMN message_chain.parent_message_id IS 'Twitch UUID of the message this is a reply to. NULL if not a reply.';
COMMENT ON COLUMN message_chain.message_text IS 'Raw message text, used to retokenize and undo Markov training on deletion.';
COMMENT ON COLUMN message_chain.is_bot_message IS 'True when the message was sent by the bot. Bot messages are not trained, so their text is skipped during deletion untraining.';

-- +goose Down
DROP TABLE message_chain;
