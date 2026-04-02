-- Seed demo data for the analytics example. Each statement is guarded by a
-- NOT EXISTS check to ensure idempotency when seeds are re-applied.

INSERT INTO products (name, category, price, created_at)
SELECT v.name, v.category, v.price::DECIMAL(10,2),
       (EXTRACT(EPOCH FROM NOW()) - 7 * 86400)::BIGINT
FROM (VALUES
  ('Espresso Machine',      'Kitchen',      299.99),
  ('Coffee Grinder',        'Kitchen',       89.95),
  ('Cast Iron Skillet',     'Kitchen',       44.50),
  ('Wireless Headphones',   'Electronics',  149.99),
  ('Mechanical Keyboard',   'Electronics',  124.00),
  ('USB-C Hub',             'Electronics',   39.99),
  ('Running Shoes',         'Sports',       119.95),
  ('Yoga Mat',              'Sports',        34.99),
  ('Water Bottle',          'Sports',        24.50),
  ('Desk Lamp',             'Office',        59.99)
) AS v(name, category, price)
WHERE NOT EXISTS (SELECT 1 FROM products LIMIT 1);

INSERT INTO orders (product_id, quantity, total, ordered_at)
SELECT p.id,
       v.quantity::INT,
       p.price * v.quantity::INT,
       (EXTRACT(EPOCH FROM NOW()) - v.days_ago * 86400)::BIGINT
FROM (VALUES
  ('Espresso Machine',      2, 6),
  ('Coffee Grinder',        3, 6),
  ('Wireless Headphones',   1, 5),
  ('Running Shoes',         2, 5),
  ('Mechanical Keyboard',   1, 4),
  ('Cast Iron Skillet',     4, 4),
  ('Yoga Mat',              2, 3),
  ('USB-C Hub',             3, 3),
  ('Espresso Machine',      1, 2),
  ('Water Bottle',          5, 2),
  ('Desk Lamp',             2, 1),
  ('Wireless Headphones',   2, 1),
  ('Coffee Grinder',        1, 0),
  ('Running Shoes',         1, 0),
  ('Mechanical Keyboard',   2, 0)
) AS v(product_name, quantity, days_ago)
JOIN products p ON p.name = v.product_name
WHERE NOT EXISTS (SELECT 1 FROM orders LIMIT 1);
