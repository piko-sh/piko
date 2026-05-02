CREATE TABLE items (
    id   UInt64,
    name String
) ENGINE = MergeTree() ORDER BY id;
