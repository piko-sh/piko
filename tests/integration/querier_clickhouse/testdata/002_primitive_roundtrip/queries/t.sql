-- piko.query(Insert, exec)
INSERT INTO t (u, i, f, s, b) VALUES ({u:UInt64}, {i:Int32}, {f:Float64}, {s:String}, {b:Bool});

-- piko.query(Get, one)
SELECT u, i, f, s, b FROM t WHERE u = {u:UInt64};
