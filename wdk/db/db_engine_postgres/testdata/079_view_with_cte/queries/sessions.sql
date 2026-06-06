-- piko.query(name: GetSessionDetails, command: one)
-- $1 as piko.param(session_id)
-- A SELECT against a view whose body starts with WITH ... ; the
-- view's column metadata must be populated even when full body
-- analysis fails, so the consumer query resolves its column refs
-- against the view.
SELECT
    session_id,
    session_account_id,
    version_status,
    version_two_factor_secret
FROM sessions_with_latest_version
WHERE session_id = $1;
