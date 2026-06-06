CREATE TABLE events (id UInt64, ts DateTime, kind String) ENGINE = MergeTree() ORDER BY ts;
