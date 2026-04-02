CREATE INDEX idx_products_attributes ON products USING GIN (attributes);
