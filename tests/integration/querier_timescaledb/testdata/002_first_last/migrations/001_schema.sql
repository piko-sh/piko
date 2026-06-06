CREATE TABLE sensor_readings (
    ts        TIMESTAMPTZ NOT NULL,
    sensor_id BIGINT      NOT NULL,
    value     DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('sensor_readings', 'ts');
