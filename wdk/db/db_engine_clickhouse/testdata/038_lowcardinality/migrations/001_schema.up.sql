CREATE TABLE lc (id UInt64, country LowCardinality(String), tier LowCardinality(Nullable(String))) ENGINE = MergeTree() ORDER BY id;
