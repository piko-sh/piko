CREATE TABLE t (id UInt64, tags Array(String)) ENGINE = MergeTree() ORDER BY id;
