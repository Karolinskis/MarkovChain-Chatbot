-- +goose Up
ALTER TABLE message_chain ADD COLUMN sender_username TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN message_chain.sender_username IS 'Twitch login of the user who sent the message.';

-- +goose Down
ALTER TABLE message_chain DROP COLUMN sender_username;
