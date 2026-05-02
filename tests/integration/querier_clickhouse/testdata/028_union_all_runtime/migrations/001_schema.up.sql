CREATE TABLE a (id UInt64, label String) ENGINE = MergeTree() ORDER BY id;
CREATE TABLE b (id UInt64, label String) ENGINE = MergeTree() ORDER BY id;
