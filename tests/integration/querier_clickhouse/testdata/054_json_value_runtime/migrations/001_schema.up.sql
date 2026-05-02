CREATE TABLE documents (
    id UInt64,
    body String
) ENGINE = MergeTree() ORDER BY id;
