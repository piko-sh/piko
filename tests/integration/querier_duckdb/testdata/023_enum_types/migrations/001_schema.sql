CREATE TYPE mood AS ENUM ('happy', 'neutral', 'sad');

CREATE TABLE people (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL,
    current_mood mood NOT NULL
);
