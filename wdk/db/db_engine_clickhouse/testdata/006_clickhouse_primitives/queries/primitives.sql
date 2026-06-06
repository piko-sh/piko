-- piko.query(GetPrimitive, one)
SELECT u8, u16, u32, u64, i8, i16, i32, i64, f32, f64, s, b, d, dt, u
FROM primitives WHERE u64 = {id:UInt64};
