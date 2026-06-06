CREATE TABLE t (id UInt64, ts DateTime, val UInt64) ENGINE = MergeTree() ORDER BY ts;
