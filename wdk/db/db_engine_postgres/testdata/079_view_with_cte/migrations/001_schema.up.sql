CREATE TABLE sessions (
    id BIGSERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE session_versions (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL,
    status TEXT NOT NULL,
    two_factor_secret TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- View body uses a WITH ... AS (SELECT DISTINCT ON ...) prefix and then
-- joins against the CTE. Reproduces the politepages identity
-- `*_with_latest_version` view shape.
CREATE OR REPLACE VIEW sessions_with_latest_version AS
WITH latest AS (
    SELECT DISTINCT ON (session_id) session_id, id
    FROM session_versions
    ORDER BY session_id, id DESC
)
SELECT
    s.id                        AS session_id,
    s.account_id                AS session_account_id,
    v.id                        AS version_id,
    v.status                    AS version_status,
    v.two_factor_secret         AS version_two_factor_secret,
    v.created_at                AS version_created_at
FROM sessions s
    JOIN latest l ON l.session_id = s.id
    JOIN session_versions v ON v.session_id = s.id AND v.id = l.id;
