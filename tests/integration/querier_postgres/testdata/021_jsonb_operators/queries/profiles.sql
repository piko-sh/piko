-- piko.query(name: ExtractStringField, command: one)
SELECT id, name, data->>'city' AS city
FROM profiles
WHERE id = $1;

-- piko.query(name: FindByContainment, command: many)
SELECT id, name
FROM profiles
WHERE data @> $1::jsonb
ORDER BY id;

-- piko.query(name: BuildObjectFromColumns, command: one)
SELECT id, jsonb_build_object('profile_name', name, 'profile_data', data) AS info
FROM profiles
WHERE id = $1;
