CREATE TABLE feeds (
    id UInt64,
    title String,
    published DateTime
) ENGINE = MergeTree() ORDER BY id;
