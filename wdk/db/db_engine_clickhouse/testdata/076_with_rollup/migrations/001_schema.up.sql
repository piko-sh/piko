CREATE TABLE t (region String, city String, val UInt64) ENGINE = MergeTree() ORDER BY (region, city);
