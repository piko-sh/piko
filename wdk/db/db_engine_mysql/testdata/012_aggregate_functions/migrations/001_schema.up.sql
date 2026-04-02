CREATE TABLE sales (
    id INT AUTO_INCREMENT PRIMARY KEY,
    product_name VARCHAR(200) NOT NULL,
    quantity INT NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    sale_date DATE NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
