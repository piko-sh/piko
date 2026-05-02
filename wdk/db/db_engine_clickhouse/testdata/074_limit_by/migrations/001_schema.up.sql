CREATE TABLE t (id UInt64, cat String, ts DateTime) ENGINE = MergeTree() ORDER BY (cat, ts);
