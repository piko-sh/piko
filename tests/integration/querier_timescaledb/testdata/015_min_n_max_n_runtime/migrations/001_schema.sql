CREATE TABLE samples (
    ts    TIMESTAMPTZ      NOT NULL,
    value DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('samples', 'ts');
