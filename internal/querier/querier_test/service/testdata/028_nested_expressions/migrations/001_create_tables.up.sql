CREATE TABLE line_items (
  id int4 PRIMARY KEY,
  quantity int2 NOT NULL,
  unit_price numeric(10,2) NOT NULL,
  tax_rate float4,
  discount numeric(10,2)
);
