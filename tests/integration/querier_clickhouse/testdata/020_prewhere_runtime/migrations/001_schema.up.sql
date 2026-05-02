CREATE TABLE events (id UInt64, region String, val UInt64) ENGINE = MergeTree() ORDER BY id;
