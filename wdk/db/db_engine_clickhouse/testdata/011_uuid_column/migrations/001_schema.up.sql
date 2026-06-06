CREATE TABLE sessions (
    id UUID,
    user_id UUID,
    created DateTime
) ENGINE = MergeTree() ORDER BY id;
