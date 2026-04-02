CREATE TABLE line_items (
  id INTEGER PRIMARY KEY,
  quantity INTEGER NOT NULL,
  unit_price REAL NOT NULL,
  tax_rate REAL,
  discount REAL
);
