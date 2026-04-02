CREATE TABLE products (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  category VARCHAR(100) NOT NULL,
  price DECIMAL(10,2) NOT NULL,
  created_at BIGINT NOT NULL
);

CREATE TABLE orders (
  id SERIAL PRIMARY KEY,
  product_id INT NOT NULL REFERENCES products(id),
  quantity INT NOT NULL CHECK (quantity > 0),
  total DECIMAL(10,2) NOT NULL,
  ordered_at BIGINT NOT NULL
);

CREATE INDEX idx_orders_product ON orders(product_id);
CREATE INDEX idx_orders_date ON orders(ordered_at);
CREATE INDEX idx_products_category ON products(category);
