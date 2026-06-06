CREATE TABLE telemetry (
    ts    TIMESTAMPTZ      NOT NULL,
    value DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('telemetry', 'ts', chunk_time_interval => INTERVAL '1 day');
