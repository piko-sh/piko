CREATE TABLE source (id UInt64, val UInt64) ENGINE = MergeTree() ORDER BY id;
CREATE TABLE dest (id UInt64, val UInt64) ENGINE = MergeTree() ORDER BY id;
