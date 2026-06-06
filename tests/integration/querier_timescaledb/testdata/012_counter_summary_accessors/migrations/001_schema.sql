CREATE TABLE counters (
    ts    TIMESTAMPTZ      NOT NULL,
    value DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('counters', 'ts');
