CREATE TABLE epochs (
    id UInt64,
    day Date32
) ENGINE = MergeTree() ORDER BY id;
