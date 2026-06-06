-- piko.query(InsertMetric, exec)
INSERT INTO metrics (id, sequence, value) VALUES ({id:UInt64}, {sequence:Int64}, {value:Int32});

-- piko.query(GetMetric, one)
SELECT id, sequence, value FROM metrics WHERE id = {id:UInt64};

-- piko.query(CountMetrics, one)
SELECT toInt64(count()) AS total FROM metrics;

-- piko.query(PruneMetricsBelow, exec)
DELETE FROM metrics WHERE id < {cutoff:UInt64};
