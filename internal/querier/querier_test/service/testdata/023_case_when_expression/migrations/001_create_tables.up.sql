CREATE TABLE orders (
  id int4 PRIMARY KEY,
  status text NOT NULL,
  amount numeric(10,2) NOT NULL,
  discount float4
);
