-- piko.query(name: ListProfiles, command: many)
SELECT id, name, details FROM profiles;

-- piko.query(name: GetStructPacked, command: many)
SELECT id, struct_pack(a := name, b := id) AS packed FROM profiles;
