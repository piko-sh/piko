CREATE DATABASE analytics;
CREATE TABLE analytics.events (
    id UInt64,
    ts DateTime
) ENGINE = MergeTree() ORDER BY (ts, id);
