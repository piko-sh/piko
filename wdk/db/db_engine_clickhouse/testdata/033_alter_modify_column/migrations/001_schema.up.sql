CREATE TABLE t (id UInt64, label String) ENGINE = MergeTree() ORDER BY id;
ALTER TABLE t MODIFY COLUMN label FixedString(32);
