CREATE TABLE events (
    id UInt64,
    user_id UInt64,
    amount UInt32
) ENGINE = MergeTree() ORDER BY id;

ALTER TABLE events ADD PROJECTION user_totals (
    SELECT user_id, sum(amount) AS total
    GROUP BY user_id
);

ALTER TABLE events MATERIALIZE PROJECTION user_totals;
