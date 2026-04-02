-- piko.name: ListProfiles
-- piko.command: many
SELECT id, name, details FROM profiles;

-- piko.name: GetStructPacked
-- piko.command: many
SELECT id, struct_pack(a := name, b := id) AS packed FROM profiles;
