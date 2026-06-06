CREATE TABLE notes (
    id UInt64,
    title String,
    deprecated String
) ENGINE = MergeTree() ORDER BY id;

ALTER TABLE notes DROP COLUMN deprecated;
