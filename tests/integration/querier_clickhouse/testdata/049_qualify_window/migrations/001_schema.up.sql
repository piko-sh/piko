CREATE TABLE scores (
    id UInt64,
    player String,
    score UInt32
) ENGINE = MergeTree() ORDER BY id;
