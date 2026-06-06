CREATE TABLE visits (
    id UInt64,
    path String,
    ts DateTime
) ENGINE = MergeTree() ORDER BY ts;
