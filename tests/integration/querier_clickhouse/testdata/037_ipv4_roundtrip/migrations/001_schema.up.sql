CREATE TABLE addresses (
    id UInt64,
    addr IPv4
) ENGINE = MergeTree() ORDER BY id;
