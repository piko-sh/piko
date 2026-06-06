-- piko.query(name: GetAccountByEmail, command: one)
-- ?1 as piko.param(email)
SELECT a.id AS account_id, av.email AS email, av.status AS status
FROM accounts a
INNER JOIN account_versions av ON av.account_id = a.id
WHERE av.email = ?1
  AND av.id = (SELECT MAX(av2.id) FROM account_versions av2 WHERE av2.account_id = a.id);

-- piko.query(name: GetAccountByEmailAtTime, command: one)
-- ?1 as piko.param(email)
-- ?2 as piko.param(before_version_id)
SELECT a.id AS account_id, av.email AS email, av.status AS status
FROM accounts a
INNER JOIN account_versions av ON av.account_id = a.id
WHERE av.email = ?1
  AND av.id = (SELECT MAX(av2.id) FROM account_versions av2 WHERE av2.account_id = a.id AND av2.id < ?2);
