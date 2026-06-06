CREATE TABLE sales (
    id UInt64,
    category String,
    amount UInt32
) ENGINE = MergeTree() ORDER BY id;
