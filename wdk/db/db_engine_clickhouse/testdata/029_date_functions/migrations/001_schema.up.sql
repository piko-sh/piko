CREATE TABLE bookings (
    id UInt64,
    ts DateTime
) ENGINE = MergeTree() ORDER BY ts;
