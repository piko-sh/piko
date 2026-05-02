CREATE TABLE sparse_readings (
    ts        TIMESTAMPTZ NOT NULL,
    sensor_id BIGINT NOT NULL,
    value     DOUBLE PRECISION
);

SELECT create_hypertable('sparse_readings', 'ts');
