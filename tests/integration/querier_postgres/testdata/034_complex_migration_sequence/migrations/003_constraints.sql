ALTER TABLE orders ADD CONSTRAINT check_total_positive CHECK (total >= 0);
ALTER TABLE orders ADD CONSTRAINT check_status_valid CHECK (status IN ('pending', 'confirmed', 'shipped', 'delivered'));
