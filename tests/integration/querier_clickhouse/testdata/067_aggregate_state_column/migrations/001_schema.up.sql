CREATE TABLE user_visits (
    user_id UInt64,
    day Date,
    uniq_state AggregateFunction(uniq, String)
) ENGINE = AggregatingMergeTree() ORDER BY (day, user_id);

INSERT INTO user_visits SELECT 1, today(), uniqState('session_a');
INSERT INTO user_visits SELECT 2, today(), uniqState('session_b');
