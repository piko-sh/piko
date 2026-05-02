CREATE TABLE t (id UInt64, old_name String) ENGINE = MergeTree() ORDER BY id;
ALTER TABLE t RENAME COLUMN old_name TO new_name;
