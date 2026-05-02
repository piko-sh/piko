CREATE TABLE integer_metrics (
    ts_epoch BIGINT           NOT NULL,
    sensor   BIGINT           NOT NULL,
    value    DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable(
    'integer_metrics',
    'ts_epoch',
    chunk_time_interval => 3600
);
