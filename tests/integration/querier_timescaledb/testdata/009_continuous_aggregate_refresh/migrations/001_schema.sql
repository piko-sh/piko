CREATE TABLE pulses (
    ts     TIMESTAMPTZ      NOT NULL,
    source BIGINT           NOT NULL,
    value  DOUBLE PRECISION
);

SELECT create_hypertable('pulses', 'ts');

CREATE MATERIALIZED VIEW hourly_pulses
WITH (timescaledb.continuous = true, timescaledb.materialized_only = false)
AS SELECT
    time_bucket('1 hour'::interval, ts) AS bucket,
    source,
    sum(value)                          AS total_value,
    count(*)                            AS sample_count
FROM pulses
GROUP BY bucket, source
WITH NO DATA;
