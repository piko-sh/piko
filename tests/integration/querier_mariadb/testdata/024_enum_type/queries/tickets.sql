-- piko.name: InsertTicket
-- piko.command: exec
INSERT INTO tickets (title, priority, status) VALUES (?, ?, ?);

-- piko.name: GetTicket
-- piko.command: one
SELECT id, title, priority, status FROM tickets WHERE id = ?;

-- piko.name: ListByPriority
-- piko.command: many
SELECT id, title, priority, status FROM tickets WHERE priority = ? ORDER BY id;

-- piko.name: ListAll
-- piko.command: many
SELECT id, title, priority, status FROM tickets ORDER BY id;
