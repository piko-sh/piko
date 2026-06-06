CREATE TABLE events (
    id UInt64,
    payload String
) ENGINE = MergeTree() ORDER BY id;
