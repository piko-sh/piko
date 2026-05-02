-- piko.query(name: ArchiveActiveAccountSessions, command: execresult)
-- ?1 as piko.param(account_id)
-- ?2 as piko.param(active)
INSERT INTO sessions_archive (account_id, session_token)
SELECT s.account_id, s.session_token
FROM sessions s
INNER JOIN accounts a ON a.id = s.account_id
WHERE s.account_id = ?1 AND a.active = ?2;

-- piko.query(name: CountArchived, command: one)
SELECT COUNT(*) AS archived_count FROM sessions_archive;

-- piko.query(name: ListArchivedTokens, command: many)
SELECT account_id, session_token FROM sessions_archive ORDER BY session_token ASC;
