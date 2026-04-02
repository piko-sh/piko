CREATE TABLE tickets (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    priority ENUM('low', 'medium', 'high') NOT NULL,
    status ENUM('open', 'in_progress', 'closed') NOT NULL DEFAULT 'open'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
