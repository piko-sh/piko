CREATE TABLE users (
    id UInt64,
    name String
) ENGINE = MergeTree() ORDER BY id;

CREATE TABLE sessions (
    id UInt64,
    user_id UInt64,
    duration UInt32
) ENGINE = MergeTree() ORDER BY id;
