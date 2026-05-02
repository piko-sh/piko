CREATE TABLE recent (
    id UInt64,
    label String
) ENGINE = MergeTree() ORDER BY id;

CREATE TABLE archived (
    id UInt64,
    label String
) ENGINE = MergeTree() ORDER BY id;
