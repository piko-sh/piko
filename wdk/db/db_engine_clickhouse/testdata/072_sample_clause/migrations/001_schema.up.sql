CREATE TABLE t (id UInt64, val UInt64) ENGINE = MergeTree() ORDER BY id SAMPLE BY id;
