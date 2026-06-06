-- piko.query(name: InsertItem, command: exec)
INSERT INTO items (id, price, quantity) VALUES (?, ?, ?);

-- piko.query(name: GetItem, command: one)
SELECT id, price, quantity, total, label FROM items WHERE id = ?;

-- piko.query(name: ListByMinTotal, command: many)
SELECT id, price, quantity, total FROM items WHERE total >= ? ORDER BY total ASC;
