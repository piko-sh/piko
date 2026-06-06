-- piko.query(name: InsertTelemetry, command: exec)
INSERT INTO telemetry (ts, value) VALUES ($1, $2);

-- piko.query(name: DropOldChunks, command: many)
WITH dropped AS (
    SELECT drop_chunks(
        'telemetry',
        older_than => '2026-01-01'::timestamptz,
        verbose    => true
    ) AS raw_chunk
)
SELECT raw_chunk::text AS dropped_chunk FROM dropped;

-- piko.query(name: RemainingRows, command: one)
SELECT count(*)::bigint AS row_count FROM telemetry;
