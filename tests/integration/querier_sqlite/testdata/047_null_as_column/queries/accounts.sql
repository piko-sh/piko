-- piko.name: GetAccountsWithPlaceholder
-- piko.command: many
SELECT id, name, email, NULL AS pending_action FROM accounts ORDER BY id;
