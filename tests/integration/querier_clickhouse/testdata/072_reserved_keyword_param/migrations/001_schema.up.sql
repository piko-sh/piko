CREATE TABLE events (
    id UInt64,
    host String
) ENGINE = MergeTree() ORDER BY id;
