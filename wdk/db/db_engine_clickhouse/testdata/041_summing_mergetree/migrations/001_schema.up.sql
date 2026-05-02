CREATE TABLE sums (id UInt64, total UInt64, count UInt64) ENGINE = SummingMergeTree((total, count)) ORDER BY id;
