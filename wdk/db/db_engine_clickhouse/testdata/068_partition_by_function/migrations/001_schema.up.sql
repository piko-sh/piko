CREATE TABLE t (id UInt64, ts DateTime) ENGINE = MergeTree() PARTITION BY toYYYYMMDD(ts) ORDER BY id;
