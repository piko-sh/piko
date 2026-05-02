CREATE TABLE t (id UInt64, raw String, sha String MATERIALIZED SHA256(raw)) ENGINE = MergeTree() ORDER BY id;
