CREATE TABLE inventory (
    id UInt64,
    counts Map(String, UInt32)
) ENGINE = MergeTree() ORDER BY id;
