-- piko.query(AllLabels, many)
SELECT id, label FROM recent
UNION ALL
SELECT id, label FROM archived
ORDER BY id;
