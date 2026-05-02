-- piko.query(name: BareReadOnly, command: many, readonly: true)
SELECT id, name FROM items;

-- piko.query(name: ExplicitReadOnlyTrue, command: many, readonly: true)
SELECT id, name FROM items;

-- piko.query(name: ExplicitReadOnlyFalse, command: many, readonly: false)
SELECT id, name FROM items;

-- piko.query(name: InsertOverriddenToReadOnly, command: exec, readonly: true)
INSERT INTO items (id, name, quantity) VALUES ($1, $2, $3);
