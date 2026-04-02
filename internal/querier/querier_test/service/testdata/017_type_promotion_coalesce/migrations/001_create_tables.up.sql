CREATE TABLE measurements (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  small_value int2 NOT NULL,
  large_value int4 NOT NULL,
  precise_value numeric(10,2)
);
