SET allow_experimental_variant_type = 1;

CREATE TABLE mixed (
    id UInt64,
    payload Variant(Int32, String)
) ENGINE = MergeTree() ORDER BY id;

INSERT INTO mixed VALUES (1, 42), (2, 'hello');
