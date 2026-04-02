-- piko.name: GetContact
-- piko.command: one
SELECT id, name, CAST(address AS VARCHAR) AS address FROM contacts WHERE id = $1;

-- piko.name: ListByCityField
-- piko.command: many
SELECT id, name, CAST(address.city AS VARCHAR) AS city FROM contacts WHERE address.city = $1 ORDER BY id;
