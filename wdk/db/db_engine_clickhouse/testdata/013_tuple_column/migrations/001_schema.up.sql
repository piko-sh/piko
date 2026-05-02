CREATE TABLE coordinates (
    id UInt64,
    point Tuple(Float64, Float64),
    named_point Tuple(x Float64, y Float64)
) ENGINE = MergeTree() ORDER BY id;
