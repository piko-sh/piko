CREATE TABLE nodes (
    id UInt64,
    parent_id UInt64
) ENGINE = MergeTree() ORDER BY id;
