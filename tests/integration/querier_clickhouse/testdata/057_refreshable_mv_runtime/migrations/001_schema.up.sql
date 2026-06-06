CREATE TABLE source_events (
    id UInt64,
    value UInt64
) ENGINE = MergeTree() ORDER BY id;

CREATE TABLE refresh_target (
    total UInt64
) ENGINE = MergeTree() ORDER BY tuple();

CREATE MATERIALIZED VIEW refresh_view
REFRESH EVERY 1 SECOND
TO refresh_target
AS SELECT sum(value) AS total
FROM source_events;
