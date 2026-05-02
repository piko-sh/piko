CREATE TABLE coords (
    id UInt64,
    position Tuple(name String, value UInt32)
) ENGINE = MergeTree() ORDER BY id;
