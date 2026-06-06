-- piko.query(name: GetAccountWithLatestVersion, command: one)
-- $1 as piko.param(account_id)
-- Verifies that columns selected from a view whose body uses a CTE
-- followed by a multi-JOIN SELECT carry the correct SQL types in the
-- generated query rather than being typed as `any`. Without the parser
-- fix that adds JOIN-introducing keywords to the expression terminator
-- set, the subsequent JOIN gets swallowed by the previous ON
-- expression, the joined table never reaches the scope chain, and the
-- view's projected columns degrade to TypeCategoryAny.
SELECT
    account_id,
    account_created_at,
    version_id,
    version_status,
    version_email
FROM accounts_with_latest
WHERE account_id = $1;
