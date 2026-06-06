CREATE TABLE t (id UInt64, payload String CODEC(LZ4)) ENGINE = MergeTree() ORDER BY id;
