CREATE TABLE accounts (
    id UInt64,
    email String,
    label Nullable(String),
    last_login Nullable(DateTime)
) ENGINE = MergeTree() ORDER BY id;
