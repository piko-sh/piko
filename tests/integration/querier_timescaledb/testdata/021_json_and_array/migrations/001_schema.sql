CREATE TABLE docs (
    id      INTEGER PRIMARY KEY,
    payload JSONB,
    tags    TEXT[]
);
