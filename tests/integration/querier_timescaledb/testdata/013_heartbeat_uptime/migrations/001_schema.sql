CREATE TABLE heartbeats (
    ts TIMESTAMPTZ NOT NULL
);

SELECT create_hypertable('heartbeats', 'ts');
