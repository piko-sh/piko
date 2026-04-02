CREATE TYPE status_type AS ENUM ('active', 'inactive', 'suspended');

CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    status status_type NOT NULL DEFAULT 'active'
);
