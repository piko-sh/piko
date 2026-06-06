CREATE TABLE country_source (
    id UInt64,
    name String
) ENGINE = MergeTree() ORDER BY id;

INSERT INTO country_source (id, name) VALUES (1, 'France'), (2, 'Spain'), (3, 'Japan');

CREATE TABLE shipments (
    id UInt64,
    country_id UInt64
) ENGINE = MergeTree() ORDER BY id;

CREATE DICTIONARY country_dict (
    id UInt64,
    name String
)
PRIMARY KEY id
SOURCE(CLICKHOUSE(USER 'default' PASSWORD 'test' TABLE 'country_source'))
LIFETIME(0)
LAYOUT(HASHED());
