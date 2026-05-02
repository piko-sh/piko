-- piko.query(name: InsertDeviceEvent, command: exec)
INSERT INTO device_events (ts, device_id) VALUES ($1, $2);

-- piko.query(name: TopDevices, command: many)
WITH agg AS (
    SELECT freq_agg(0.05::double precision, device_id) AS state
    FROM device_events
)
SELECT
    value::bigint                    AS device_id,
    max_frequency(state, value)::double precision AS max_freq,
    min_frequency(state, value)::double precision AS min_freq
FROM agg, topn(state, 5) AS value;
