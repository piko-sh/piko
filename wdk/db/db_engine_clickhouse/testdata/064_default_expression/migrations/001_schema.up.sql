CREATE TABLE t (id UInt64, created DateTime DEFAULT now()) ENGINE = MergeTree() ORDER BY id;
