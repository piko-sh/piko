-- piko.name: GetDoubledPrice
-- piko.command: one
SELECT double_price(price) AS doubled FROM products WHERE id = $1;

-- piko.name: GetCurrentPrice
-- piko.command: one
SELECT get_current_price($1::integer) AS price;
