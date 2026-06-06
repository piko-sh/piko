CREATE TABLE sales (id UInt64, region String, amount UInt64) ENGINE = MergeTree() ORDER BY id;
