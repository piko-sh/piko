CREATE TABLE events (id UInt64, country LowCardinality(String)) ENGINE = MergeTree() ORDER BY id;
