CREATE TABLE readings (
    ts        TIMESTAMPTZ      NOT NULL,
    device_id BIGINT           NOT NULL,
    value     DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('readings', 'ts');

CREATE INDEX readings_ts_device_idx ON readings (ts, device_id);
