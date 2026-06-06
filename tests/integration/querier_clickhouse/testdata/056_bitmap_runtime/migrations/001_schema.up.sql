CREATE TABLE raw_visitors (
    day Date,
    user_id UInt64
) ENGINE = MergeTree() ORDER BY day;

CREATE TABLE bitmap_visitors (
    day Date,
    visitors AggregateFunction(groupBitmap, UInt64)
) ENGINE = AggregatingMergeTree() ORDER BY day;
