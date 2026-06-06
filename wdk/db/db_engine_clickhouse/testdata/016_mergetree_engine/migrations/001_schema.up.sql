CREATE TABLE logs (
    id UInt64,
    ts DateTime,
    level String,
    message String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(ts)
ORDER BY (ts, id)
SETTINGS index_granularity = 8192;
