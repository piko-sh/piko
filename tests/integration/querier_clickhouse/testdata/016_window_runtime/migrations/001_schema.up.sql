CREATE TABLE scores (player String, score UInt64) ENGINE = MergeTree() ORDER BY player;
