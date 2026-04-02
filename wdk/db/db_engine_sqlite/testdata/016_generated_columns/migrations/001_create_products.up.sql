CREATE TABLE products (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  price REAL NOT NULL,
  quantity INTEGER NOT NULL,
  total_value REAL GENERATED ALWAYS AS (price * quantity) STORED,
  display_name TEXT AS (upper(name)) VIRTUAL
);
