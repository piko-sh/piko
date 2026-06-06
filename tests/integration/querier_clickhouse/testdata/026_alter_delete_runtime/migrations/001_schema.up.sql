CREATE TABLE entries (id UInt64, tombstoned UInt8) ENGINE = MergeTree() ORDER BY id;
