-- piko.name: ListAllMessages
-- piko.command: many
SELECT id, subject, received_date AS message_date, 'inbox' AS source
FROM inbox
UNION ALL
SELECT id, subject, sent_date AS message_date, 'sent' AS source
FROM sent
ORDER BY message_date;
