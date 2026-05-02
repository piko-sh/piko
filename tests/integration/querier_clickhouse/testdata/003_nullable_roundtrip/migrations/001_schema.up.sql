CREATE TABLE accounts (
    id UInt64,
    label Nullable(String)
) ENGINE = MergeTree() ORDER BY id;
