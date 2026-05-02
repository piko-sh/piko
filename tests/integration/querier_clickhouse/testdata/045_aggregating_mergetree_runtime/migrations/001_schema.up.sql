CREATE TABLE raw_views (
    page_id UInt64,
    visitor String,
    views UInt64
) ENGINE = MergeTree() ORDER BY page_id;

CREATE TABLE page_metrics (
    page_id UInt64,
    visitor_state AggregateFunction(uniq, String),
    view_total AggregateFunction(sum, UInt64)
) ENGINE = AggregatingMergeTree() ORDER BY page_id;
