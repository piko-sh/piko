CREATE TABLE agg (key UInt64, val UInt64) ENGINE = AggregatingMergeTree() ORDER BY key;
