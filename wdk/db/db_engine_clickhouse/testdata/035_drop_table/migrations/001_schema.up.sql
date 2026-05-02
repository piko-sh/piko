CREATE TABLE keep (id UInt64) ENGINE = MergeTree() ORDER BY id;
CREATE TABLE remove_me (id UInt64) ENGINE = MergeTree() ORDER BY id;
DROP TABLE remove_me;
