CREATE TABLE counters (key UInt64, count UInt64) ENGINE = SummingMergeTree((count)) ORDER BY key;
