CREATE TABLE t (id UInt64, x String) ENGINE = MergeTree() ORDER BY id;
CREATE VIEW v AS SELECT id, x FROM t;
