CREATE TABLE events (id UInt64, tags Array(String)) ENGINE = MergeTree() ORDER BY id;
