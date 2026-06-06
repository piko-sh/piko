CREATE TABLE sales (
    id UInt64,
    region String,
    amount Decimal(18, 2)
) ENGINE = MergeTree() ORDER BY id;
