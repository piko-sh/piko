CREATE TABLE labelled (
    id UInt64,
    metadata Map(String, String),
    counters Map(String, UInt64)
) ENGINE = MergeTree() ORDER BY id;
