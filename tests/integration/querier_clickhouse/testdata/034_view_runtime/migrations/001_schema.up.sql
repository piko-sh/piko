CREATE TABLE base (id UInt64, val UInt64) ENGINE = MergeTree() ORDER BY id;
CREATE VIEW doubled AS SELECT id, val * 2 AS twice FROM base;
