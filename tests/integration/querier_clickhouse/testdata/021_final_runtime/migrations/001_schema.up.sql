CREATE TABLE versioned (id UInt64, val UInt64, version UInt64) ENGINE = ReplacingMergeTree(version) ORDER BY id;
