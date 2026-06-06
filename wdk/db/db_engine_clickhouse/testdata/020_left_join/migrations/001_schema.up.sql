CREATE TABLE products (
    id UInt64,
    name String
) ENGINE = MergeTree() ORDER BY id;

CREATE TABLE reviews (
    product_id UInt64,
    stars UInt8
) ENGINE = MergeTree() ORDER BY product_id;
