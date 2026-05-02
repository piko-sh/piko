CREATE TABLE hashes (id UInt64, sha FixedString(20)) ENGINE = MergeTree() ORDER BY id;
