CREATE TABLE prices (
    id UInt64,
    amount Decimal128(18)
) ENGINE = MergeTree() ORDER BY id;
