CREATE TABLE prices (id UInt64, amount Decimal(18, 4)) ENGINE = MergeTree() ORDER BY id;
