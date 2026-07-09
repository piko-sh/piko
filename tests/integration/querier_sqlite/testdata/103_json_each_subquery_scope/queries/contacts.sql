-- piko.query(name: SearchContactsExists, command: many)
SELECT id
FROM customers
WHERE EXISTS (
    SELECT 1 FROM json_each(company_contacts) je
    WHERE json_extract(je.value, '$.name') LIKE ('%' || ? || '%')
)
ORDER BY id;

-- piko.query(name: SearchContactsScalar, command: many)
SELECT
    id,
    (SELECT json_extract(je.value, '$.name')
       FROM json_each(company_contacts) je
       WHERE json_extract(je.value, '$.uuid') = ?) AS matched_name
FROM customers
ORDER BY id;
