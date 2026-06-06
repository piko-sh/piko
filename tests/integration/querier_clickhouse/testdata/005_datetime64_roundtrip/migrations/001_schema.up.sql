CREATE TABLE events (
    id UInt64,
    occurred_at DateTime64(6, 'UTC')
) ENGINE = MergeTree() ORDER BY id;
