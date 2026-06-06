CREATE TABLE orders (
    id UInt64,
    category String,
    day Date,
    amount UInt64
) ENGINE = MergeTree() ORDER BY id;
