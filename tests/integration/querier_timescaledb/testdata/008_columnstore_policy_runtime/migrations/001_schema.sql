CREATE TABLE metrics (
    ts     TIMESTAMPTZ      NOT NULL,
    device TEXT             NOT NULL,
    value  DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('metrics', 'ts', chunk_time_interval => INTERVAL '1 day');

ALTER TABLE metrics SET (
    timescaledb.enable_columnstore = true,
    timescaledb.segmentby = 'device',
    timescaledb.orderby = 'ts DESC'
);
