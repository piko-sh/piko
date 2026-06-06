CREATE TABLE tasks (
    id UInt64,
    status Enum8('pending' = 1, 'running' = 2, 'done' = 3, 'failed' = 4)
) ENGINE = MergeTree() ORDER BY id;
