CREATE TABLE nested_tbl (id UInt64, items Array(Tuple(String, UInt64))) ENGINE = MergeTree() ORDER BY id;
