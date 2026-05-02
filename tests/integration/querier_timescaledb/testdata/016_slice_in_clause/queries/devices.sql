-- piko.query(name: InsertDevice, command: exec)
INSERT INTO devices (id, name) VALUES ($1, $2);

-- piko.query(name: GetDevicesByIDs, command: many)
-- $1 as piko.param(ids, kind: slice)
SELECT id, name FROM devices WHERE id IN ($1) ORDER BY id;
