CREATE TABLE events (id UInt64, kind String) ENGINE = MergeTree() ORDER BY id;
