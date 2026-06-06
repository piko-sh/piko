CREATE EXTENSION IF NOT EXISTS timescaledb_toolkit CASCADE;

CREATE TABLE signal (
    ts    TIMESTAMPTZ      NOT NULL,
    value DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('signal', 'ts');
