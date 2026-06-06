CREATE TABLE measurements (
    ts        TIMESTAMPTZ NOT NULL,
    group_id  BIGINT NOT NULL,
    sample    DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('measurements', 'ts');
