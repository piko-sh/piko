CREATE TABLE customers (
    id UInt64,
    name String
) ENGINE = MergeTree() ORDER BY id;

CREATE TABLE orders (
    id UInt64,
    customer_id UInt64,
    amount UInt32
) ENGINE = MergeTree() ORDER BY id;
