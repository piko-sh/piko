CREATE TABLE readings (
    ts          TIMESTAMPTZ NOT NULL,
    device_id   BIGINT      NOT NULL,
    temperature DOUBLE PRECISION
);

SELECT create_hypertable('readings', 'ts');
