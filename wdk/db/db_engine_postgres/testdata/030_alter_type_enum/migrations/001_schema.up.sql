CREATE TYPE colour AS ENUM ('red', 'green', 'blue');
CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    colour colour NOT NULL
);
