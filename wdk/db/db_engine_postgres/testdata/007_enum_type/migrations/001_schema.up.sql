CREATE TYPE status AS ENUM ('active', 'inactive', 'pending');
CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    status status NOT NULL DEFAULT 'pending'
);
