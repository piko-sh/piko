CREATE TABLE configurations (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL,
    settings MAP(VARCHAR, VARCHAR) NOT NULL
);
