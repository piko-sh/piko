-- piko.query(name: InsertHeartbeat, command: exec)
INSERT INTO heartbeats (ts) VALUES ($1);

-- piko.query(name: HeartbeatAccessors, command: one)
WITH summary AS (
    SELECT heartbeat_agg(
        ts,
        '2026-01-01T00:00:00Z'::timestamptz,
        '10 minutes'::interval,
        '5 minutes'::interval
    ) AS state
    FROM heartbeats
)
SELECT
    (uptime(state))::text         AS uptime_text,
    (downtime(state))::text       AS downtime_text,
    (num_live_ranges(state))      AS live_range_count
FROM summary;
