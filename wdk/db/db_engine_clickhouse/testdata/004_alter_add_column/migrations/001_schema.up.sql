CREATE TABLE items (
    id UInt64
) ENGINE = MergeTree() ORDER BY id;

ALTER TABLE items ADD COLUMN name String;
ALTER TABLE items ADD COLUMN active Bool DEFAULT true;
