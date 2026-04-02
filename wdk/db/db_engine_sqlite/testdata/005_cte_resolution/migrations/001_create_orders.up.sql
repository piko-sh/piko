CREATE TABLE orders (
  id INTEGER PRIMARY KEY,
  customer_name TEXT NOT NULL,
  amount REAL NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending'
);
