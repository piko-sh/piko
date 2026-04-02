-- piko.name: GetDoubledPrice
-- piko.command: one
SELECT id, double_price(price) AS doubled FROM products WHERE id = ?;

-- piko.name: GetPriceWithTax
-- piko.command: one
SELECT id, add_tax(price, 20.00) AS with_tax FROM products WHERE id = ?;
