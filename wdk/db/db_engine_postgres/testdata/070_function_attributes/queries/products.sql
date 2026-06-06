-- piko.query(name: GetDoubledPrice, command: one)
SELECT double_price(price) AS doubled FROM products WHERE id = $1;

-- piko.query(name: GetCurrentPrice, command: one)
SELECT get_current_price($1::integer) AS price;
