CREATE TABLE events (
    id UInt64,
    ts DateTime,
    payload String
) ENGINE = MergeTree() ORDER BY (ts, id);
