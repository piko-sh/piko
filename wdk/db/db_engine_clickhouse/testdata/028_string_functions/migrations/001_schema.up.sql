CREATE TABLE handles (
    id UInt64,
    name String,
    domain String
) ENGINE = MergeTree() ORDER BY id;
