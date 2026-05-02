CREATE TABLE signed_big (id UInt64, big_neg Int128, huge_neg Int256) ENGINE = MergeTree() ORDER BY id;
