-- piko.query(name: InsertSignal, command: exec)
INSERT INTO signal (ts, value) VALUES ($1, $2);

-- piko.query(name: BucketCount, command: one)
WITH agg AS (
    SELECT lttb(ts, value, 10) AS state
    FROM signal
)
SELECT count(*)::bigint AS bucket_count
FROM agg, unnest(state);

-- piko.query(name: SampleCount, command: one)
SELECT count(*)::bigint AS sample_count FROM signal;
