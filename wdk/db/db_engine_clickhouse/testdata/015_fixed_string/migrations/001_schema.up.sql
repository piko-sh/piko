CREATE TABLE hashes (
    id UInt64,
    sha1 FixedString(20),
    sha256 FixedString(32)
) ENGINE = MergeTree() ORDER BY id;
