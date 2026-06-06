-- piko.query(name: InsertTicket, command: exec)
INSERT INTO tickets (title, priority, status) VALUES (?, ?, ?);

-- piko.query(name: GetTicket, command: one)
SELECT id, title, priority, status FROM tickets WHERE id = ?;

-- piko.query(name: ListByPriority, command: many)
SELECT id, title, priority, status FROM tickets WHERE priority = ? ORDER BY id;

-- piko.query(name: ListAll, command: many)
SELECT id, title, priority, status FROM tickets ORDER BY id;
