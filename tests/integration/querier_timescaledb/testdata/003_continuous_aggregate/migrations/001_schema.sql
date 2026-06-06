CREATE TABLE temperatures (
    ts          TIMESTAMPTZ NOT NULL,
    location_id BIGINT      NOT NULL,
    temperature DOUBLE PRECISION
);

SELECT create_hypertable('temperatures', 'ts');

CREATE MATERIALIZED VIEW hourly_temperatures
WITH (timescaledb.continuous = true, timescaledb.materialized_only = false)
AS SELECT
    time_bucket('1 hour'::interval, ts) AS bucket,
    location_id,
    avg(temperature) AS mean_temperature,
    count(*) AS sample_count
FROM temperatures
GROUP BY bucket, location_id
WITH NO DATA;
