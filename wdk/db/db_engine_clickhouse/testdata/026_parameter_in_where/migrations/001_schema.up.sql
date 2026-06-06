CREATE TABLE entries (
    id UInt64,
    category String,
    score Float64
) ENGINE = MergeTree() ORDER BY id;
