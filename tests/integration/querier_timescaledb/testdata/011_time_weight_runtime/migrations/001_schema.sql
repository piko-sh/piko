CREATE TABLE gauge_samples (
    ts    TIMESTAMPTZ      NOT NULL,
    value DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('gauge_samples', 'ts');
