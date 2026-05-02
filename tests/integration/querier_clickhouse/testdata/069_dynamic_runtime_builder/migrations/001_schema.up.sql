CREATE TABLE products (
    id UInt64,
    name String,
    price UInt32,
    in_stock Bool
) ENGINE = MergeTree() ORDER BY id;
