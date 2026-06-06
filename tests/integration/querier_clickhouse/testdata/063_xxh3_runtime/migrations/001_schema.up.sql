CREATE TABLE payloads (
    id UInt64,
    body String
) ENGINE = MergeTree() ORDER BY id;
