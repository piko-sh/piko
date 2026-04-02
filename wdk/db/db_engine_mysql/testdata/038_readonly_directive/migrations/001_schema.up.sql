CREATE TABLE items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    quantity INT NOT NULL
);

DELIMITER //

CREATE FUNCTION volatile_func(val INT) RETURNS INT
MODIFIES SQL DATA
BEGIN
    RETURN val * 2;
END //

-- piko.readonly
CREATE FUNCTION overridden_readonly_func(val INT) RETURNS INT
MODIFIES SQL DATA
BEGIN
    RETURN val * 2;
END //

-- piko.readonly(false)
CREATE FUNCTION overridden_not_readonly_func(val INT) RETURNS INT
DETERMINISTIC NO SQL
BEGIN
    RETURN val * 2;
END //

DELIMITER ;
