CREATE TABLE keyed (id UInt64, payload String, version UInt64) ENGINE = ReplacingMergeTree(version) ORDER BY id;
