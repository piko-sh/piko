CREATE TABLE t (
    u UInt64,
    i Int32,
    f Float64,
    s String,
    b Bool
) ENGINE = MergeTree() ORDER BY u;
