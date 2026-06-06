CREATE TABLE noop (
    id UInt64,
    value String
) ENGINE = MergeTree() ORDER BY id;
