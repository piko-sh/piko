CREATE TABLE profiles (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    data JSONB NOT NULL
);
