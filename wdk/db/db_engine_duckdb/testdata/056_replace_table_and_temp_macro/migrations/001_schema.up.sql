CREATE TABLE events (
    id BIGINT PRIMARY KEY,
    slug VARCHAR
);

CREATE OR REPLACE TABLE events (
    id BIGINT PRIMARY KEY,
    slug VARCHAR NOT NULL,
    payload VARCHAR
);

CREATE TEMP MACRO bump(value, step := 1) AS value + step;
