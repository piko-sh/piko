-- piko.query(name: GetLineTotals, command: many)
SELECT id, quantity * unit_price AS total FROM line_items;
