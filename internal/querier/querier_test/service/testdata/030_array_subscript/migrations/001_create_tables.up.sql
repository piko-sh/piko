CREATE TABLE posts (
  id serial PRIMARY KEY,
  title text NOT NULL,
  tags text[] NOT NULL
);
