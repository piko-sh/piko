CREATE TABLE tree (id UInt64, parent_id Nullable(UInt64)) ENGINE = MergeTree() ORDER BY id;
