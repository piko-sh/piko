CREATE TYPE status AS ENUM ('active', 'inactive', 'pending');

CREATE TABLE items (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL,
    status status NOT NULL DEFAULT 'pending'
);
