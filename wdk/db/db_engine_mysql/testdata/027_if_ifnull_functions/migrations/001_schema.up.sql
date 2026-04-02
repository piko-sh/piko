CREATE TABLE inventory (
    id INT AUTO_INCREMENT PRIMARY KEY,
    item_name VARCHAR(100) NOT NULL,
    quantity INT NOT NULL,
    reorder_level INT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
