CREATE TABLE accounts (
    id uuid PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE account_versions (
    id uuid PRIMARY KEY,
    account_id uuid NOT NULL,
    status TEXT NOT NULL,
    email TEXT NOT NULL
);

-- View body uses a CTE followed by a multi-JOIN SELECT that projects
-- columns from the JOINed tables. Before the fix, the parser's
-- expression-terminator set did not include JOIN-introducing keywords,
-- so the ON expression of the first JOIN swallowed the subsequent
-- JOIN clause. That dropped the second JOIN (account_versions v) from
-- the parsed FromTables/JoinClauses, leaving the scope chain unaware
-- of `v`, and the type resolver fell back to TypeCategoryAny (18) for
-- every column projected from `v.*`. Downstream queries that SELECT
-- those columns from this view then also inherited `any` types,
-- breaking compile-time type safety in the generated DAL.
CREATE VIEW accounts_with_latest AS
WITH latest AS (
    SELECT DISTINCT ON (account_id) account_id, id
    FROM account_versions
    ORDER BY account_id, id DESC
)
SELECT
    u.id              AS account_id,
    u.created_at      AS account_created_at,
    v.id              AS version_id,
    v.status          AS version_status,
    v.email           AS version_email
FROM accounts u
    JOIN latest l ON l.account_id = u.id
    JOIN account_versions v ON v.account_id = u.id AND v.id = l.id;
