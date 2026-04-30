CREATE TABLE orders (
  id int4 PRIMARY KEY,
  total int4 NOT NULL
);

CREATE TABLE accounts (
  id int4 PRIMARY KEY,
  email text NOT NULL
);

CREATE TABLE customers (
  id int4 PRIMARY KEY,
  email text NOT NULL
);
