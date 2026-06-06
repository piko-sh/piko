CREATE TABLE left_t (id UInt64) ENGINE = MergeTree() ORDER BY id;
CREATE TABLE right_t (id UInt64) ENGINE = MergeTree() ORDER BY id;
