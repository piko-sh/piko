CREATE TABLE documents (
    id INTEGER PRIMARY KEY,
    title VARCHAR NOT NULL,
    tags LIST(VARCHAR) NOT NULL,
    scores LIST(INTEGER)
);
