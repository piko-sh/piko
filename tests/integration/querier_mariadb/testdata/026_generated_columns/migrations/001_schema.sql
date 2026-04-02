CREATE TABLE line_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    product VARCHAR(255) NOT NULL,
    quantity INT NOT NULL,
    unit_price INT NOT NULL,
    total_price INT GENERATED ALWAYS AS (quantity * unit_price) STORED,
    discount_pct INT NOT NULL DEFAULT 0,
    discounted_price INT GENERATED ALWAYS AS (quantity * unit_price * (100 - discount_pct) / 100) STORED
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
