CREATE TABLE users (
    id UInt64,
    email String
) ENGINE = MergeTree() ORDER BY id;

CREATE TABLE accounts (
    id UInt64,
    user_id UInt64,
    balance Decimal(18, 4)
) ENGINE = MergeTree() ORDER BY id;
