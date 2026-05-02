CREATE TABLE prices (
    id UInt64,
    amount Decimal(18, 4),
    fee Decimal64(2),
    micro_amount Decimal128(8)
) ENGINE = MergeTree() ORDER BY id;
