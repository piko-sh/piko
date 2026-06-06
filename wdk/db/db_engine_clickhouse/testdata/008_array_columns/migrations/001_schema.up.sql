CREATE TABLE products (
    id UInt64,
    tags Array(String),
    scores Array(Float64),
    optional_codes Array(Nullable(UInt32))
) ENGINE = MergeTree() ORDER BY id;
