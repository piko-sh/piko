CREATE TABLE timed (
    id UInt64,
    micro_ts DateTime64(6),
    nano_ts DateTime64(9, 'UTC')
) ENGINE = MergeTree() ORDER BY id;
