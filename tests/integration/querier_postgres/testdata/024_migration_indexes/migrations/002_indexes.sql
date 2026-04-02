CREATE INDEX idx_products_category ON products (category);
CREATE UNIQUE INDEX idx_products_sku ON products (sku);
CREATE INDEX idx_products_active_price ON products (price) WHERE active = true;
