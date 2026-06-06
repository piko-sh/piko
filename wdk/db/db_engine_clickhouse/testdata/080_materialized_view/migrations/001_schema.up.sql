CREATE TABLE src (id UInt64, val UInt64) ENGINE = MergeTree() ORDER BY id;
CREATE MATERIALIZED VIEW mv ENGINE = MergeTree() ORDER BY id AS SELECT id, val * 2 AS doubled FROM src;
