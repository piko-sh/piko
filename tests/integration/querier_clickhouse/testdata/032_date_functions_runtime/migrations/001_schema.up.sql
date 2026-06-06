CREATE TABLE events (id UInt64, ts DateTime) ENGINE = MergeTree() ORDER BY ts;
