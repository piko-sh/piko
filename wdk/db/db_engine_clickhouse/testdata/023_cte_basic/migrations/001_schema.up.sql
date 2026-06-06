CREATE TABLE orders (
    id UInt64,
    user_id UInt64,
    total Decimal(18, 2)
) ENGINE = MergeTree() ORDER BY id;
