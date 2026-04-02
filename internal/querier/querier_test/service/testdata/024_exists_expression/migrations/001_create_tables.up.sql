CREATE TABLE users (
  id int4 PRIMARY KEY,
  name text NOT NULL
);

CREATE TABLE orders (
  id int4 PRIMARY KEY,
  user_id int4 NOT NULL
);
