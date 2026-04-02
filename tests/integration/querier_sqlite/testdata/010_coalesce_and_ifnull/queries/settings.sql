-- piko.name: GetSettingValue
-- piko.command: one
SELECT
    id,
    COALESCE(user_value, default_value) AS effective_value,
    COALESCE(user_value, 'fallback') AS with_literal_fallback
FROM settings
WHERE id = ?;
