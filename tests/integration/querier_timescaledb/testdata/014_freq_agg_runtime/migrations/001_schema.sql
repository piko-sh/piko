CREATE EXTENSION IF NOT EXISTS timescaledb_toolkit CASCADE;

CREATE TABLE device_events (
    ts        TIMESTAMPTZ NOT NULL,
    device_id BIGINT      NOT NULL
);

SELECT create_hypertable('device_events', 'ts');
