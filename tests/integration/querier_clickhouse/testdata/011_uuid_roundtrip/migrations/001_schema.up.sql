CREATE TABLE sessions (id UUID, label String) ENGINE = MergeTree() ORDER BY id;
