-- piko.name: ReconcileBalances
-- piko.command: many
WITH account_totals AS (
    SELECT account_id, SUM(amount) AS total
    FROM transactions
    GROUP BY account_id
)
UPDATE accounts SET balance = at.total
FROM account_totals at
WHERE accounts.id = at.account_id
RETURNING accounts.id, accounts.balance;
