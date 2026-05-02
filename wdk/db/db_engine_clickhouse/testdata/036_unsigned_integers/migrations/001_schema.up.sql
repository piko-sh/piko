CREATE TABLE big (id UInt64, big_id UInt128, huge_id UInt256) ENGINE = MergeTree() ORDER BY id;
