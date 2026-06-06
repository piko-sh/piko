-- piko.query(name: GetLatestFolderVersion, command: one)
-- ?1 as piko.param(before_version_id)
-- ?2 as piko.param(folder_id)
WITH latest AS (
    SELECT MAX(id) AS id
    FROM content_media_folder_versions
    WHERE media_folder_id = ?2 AND id < ?1
)
SELECT m.id AS folder_id, v.id AS version_id, v.status AS status
FROM content_media_folders m
INNER JOIN content_media_folder_versions v ON v.media_folder_id = m.id
INNER JOIN latest l ON v.id = l.id
WHERE m.id = ?2;
