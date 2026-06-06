CREATE TABLE t (id UInt64, payload String TTL toDateTime(id) + INTERVAL 1 MONTH) ENGINE = MergeTree() ORDER BY id;
