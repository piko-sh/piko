CREATE TABLE events (
    id UInt64,
    category String,
    amount UInt32,
    user_id UInt64
) ENGINE = MergeTree() ORDER BY id;
