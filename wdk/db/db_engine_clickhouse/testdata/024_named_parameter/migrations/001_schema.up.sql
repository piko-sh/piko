CREATE TABLE customers (
    id UInt64,
    email String,
    country String
) ENGINE = MergeTree() ORDER BY id;
