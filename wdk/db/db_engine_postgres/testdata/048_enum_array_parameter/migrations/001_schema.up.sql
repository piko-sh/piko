CREATE TYPE colour AS ENUM ('red', 'green', 'blue');
CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    colour colour NOT NULL
);
