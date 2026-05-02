CREATE TABLE events (
    id UInt64,
    ts DateTime,
    amount UInt32
) ENGINE = MergeTree() ORDER BY ts;
