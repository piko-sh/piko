CREATE TABLE logs (ts DateTime, level String, msg String) ENGINE = MergeTree() PARTITION BY toYYYYMM(ts) ORDER BY ts;
