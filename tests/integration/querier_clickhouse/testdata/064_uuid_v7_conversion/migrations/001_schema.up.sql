CREATE TABLE uuid_log (
    id UUID,
    label String
) ENGINE = MergeTree() ORDER BY id;
