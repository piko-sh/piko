CREATE UNIQUE INDEX idx_customers_email ON customers (email);
CREATE INDEX idx_orders_customer ON orders (customer_id);
CREATE INDEX idx_orders_status ON orders (status);
CREATE INDEX idx_customers_metadata ON customers USING GIN (metadata);
