CREATE TABLE t (id UInt64) ENGINE = MergeTree() ORDER BY id SETTINGS index_granularity = 4096, merge_max_block_size = 8192;
