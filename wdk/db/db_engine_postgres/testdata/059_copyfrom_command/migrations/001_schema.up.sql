CREATE TABLE measurements (
    id SERIAL PRIMARY KEY,
    sensor_id INTEGER NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL
);
