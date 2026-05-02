CREATE TABLE metrics (
    id UInt64,
    value UInt32
) ENGINE = MergeTree() ORDER BY id;

ALTER TABLE metrics ADD INDEX idx_value value TYPE minmax GRANULARITY 4;
