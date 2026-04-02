CREATE SEQUENCE IF NOT EXISTS sales_id_seq;

CREATE TABLE IF NOT EXISTS sales (
  id INTEGER PRIMARY KEY DEFAULT nextval('sales_id_seq'),
  product VARCHAR NOT NULL,
  category VARCHAR NOT NULL,
  region VARCHAR NOT NULL,
  amount DECIMAL(10, 2) NOT NULL,
  quantity INTEGER NOT NULL,
  sold_at TIMESTAMP NOT NULL
);

-- Seed with sample analytics data spanning several months.
INSERT INTO sales (product, category, region, amount, quantity, sold_at) VALUES
  ('Laptop Pro 15',    'Electronics', 'Europe',        1299.99, 3,  '2026-01-05 09:14:00'),
  ('Wireless Mouse',   'Electronics', 'North America', 29.99,   12, '2026-01-07 14:30:00'),
  ('Standing Desk',    'Furniture',   'Europe',        549.00,  2,  '2026-01-12 11:00:00'),
  ('USB-C Hub',        'Electronics', 'Asia Pacific',  59.99,   8,  '2026-01-15 16:45:00'),
  ('Ergonomic Chair',  'Furniture',   'North America', 399.00,  5,  '2026-01-20 10:20:00'),
  ('Mechanical KB',    'Electronics', 'Europe',        149.99,  7,  '2026-02-02 08:30:00'),
  ('Monitor 27"',      'Electronics', 'North America', 449.99,  4,  '2026-02-05 13:15:00'),
  ('Desk Lamp',        'Furniture',   'Asia Pacific',  79.99,   10, '2026-02-10 09:00:00'),
  ('Laptop Pro 15',    'Electronics', 'North America', 1299.99, 2,  '2026-02-14 15:30:00'),
  ('Webcam HD',        'Electronics', 'Europe',        89.99,   6,  '2026-02-18 11:45:00'),
  ('Standing Desk',    'Furniture',   'North America', 549.00,  3,  '2026-02-22 14:00:00'),
  ('Wireless Mouse',   'Electronics', 'Asia Pacific',  29.99,   15, '2026-03-01 10:00:00'),
  ('Ergonomic Chair',  'Furniture',   'Europe',        399.00,  4,  '2026-03-05 09:30:00'),
  ('USB-C Hub',        'Electronics', 'North America', 59.99,   11, '2026-03-08 16:00:00'),
  ('Laptop Pro 15',    'Electronics', 'Asia Pacific',  1299.99, 1,  '2026-03-12 08:15:00'),
  ('Monitor 27"',      'Electronics', 'Europe',        449.99,  3,  '2026-03-15 12:30:00'),
  ('Mechanical KB',    'Electronics', 'North America', 149.99,  9,  '2026-03-18 14:45:00'),
  ('Desk Lamp',        'Furniture',   'North America', 79.99,   8,  '2026-03-22 11:00:00'),
  ('Webcam HD',        'Electronics', 'Asia Pacific',  89.99,   5,  '2026-03-25 15:15:00'),
  ('Standing Desk',    'Furniture',   'Asia Pacific',  549.00,  2,  '2026-03-28 10:30:00');
