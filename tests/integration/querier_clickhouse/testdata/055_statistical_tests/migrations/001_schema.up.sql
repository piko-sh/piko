CREATE TABLE samples (
    cohort String,
    value Float64
) ENGINE = MergeTree() ORDER BY cohort;
