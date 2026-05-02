CREATE TABLE primitives (
    u8 UInt8,
    u16 UInt16,
    u32 UInt32,
    u64 UInt64,
    i8 Int8,
    i16 Int16,
    i32 Int32,
    i64 Int64,
    f32 Float32,
    f64 Float64,
    s String,
    b Bool,
    d Date,
    dt DateTime,
    u UUID
) ENGINE = MergeTree() ORDER BY u64;
