CREATE TABLE chunk_events (
    ts        TIMESTAMPTZ NOT NULL,
    event_id  BIGINT NOT NULL,
    payload   TEXT
);

SELECT create_hypertable('chunk_events', 'ts', chunk_time_interval => INTERVAL '1 day');
