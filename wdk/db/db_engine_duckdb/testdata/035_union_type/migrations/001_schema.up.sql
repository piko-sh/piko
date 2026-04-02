CREATE TABLE events (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL,
    payload UNION(text_value VARCHAR, int_value INTEGER, bool_value BOOLEAN)
);
