INSTALL httpfs;
LOAD httpfs;

CREATE TABLE files (
    id INTEGER PRIMARY KEY,
    path VARCHAR NOT NULL
);
