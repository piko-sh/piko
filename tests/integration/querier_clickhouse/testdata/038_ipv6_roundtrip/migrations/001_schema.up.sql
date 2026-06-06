CREATE TABLE addresses (
    id UInt64,
    addr IPv6
) ENGINE = MergeTree() ORDER BY id;
