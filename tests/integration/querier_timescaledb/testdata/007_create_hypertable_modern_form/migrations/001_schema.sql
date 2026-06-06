CREATE TABLE telemetry (
    ts        TIMESTAMPTZ NOT NULL,
    device_id BIGINT      NOT NULL,
    value     DOUBLE PRECISION
)
WITH (
    tsdb.hypertable,
    tsdb.partition_column = 'ts',
    tsdb.chunk_interval = '1 day'
);
