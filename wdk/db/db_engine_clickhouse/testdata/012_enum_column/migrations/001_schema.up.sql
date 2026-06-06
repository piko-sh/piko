CREATE TABLE tickets (
    id UInt64,
    status Enum8('open' = 1, 'closed' = 2, 'archived' = 3),
    priority Enum16('low' = -1, 'medium' = 0, 'high' = 1)
) ENGINE = MergeTree() ORDER BY id;
