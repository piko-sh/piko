CREATE TABLE metrics (
    id       UInt64,
    sequence Int64,
    value    Int32
) ENGINE = MergeTree() ORDER BY id;
