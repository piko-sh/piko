-- piko.query(name: ClaimRelease, command: one, optional: true)
INSERT INTO release_lease (release_id, publish_digest, state, first_seen_at, heartbeat_at)
VALUES (?, ?, 'publishing', ?, ?)
ON CONFLICT(release_id) DO NOTHING
RETURNING release_id;

-- piko.query(name: GetRelease, command: one, optional: true)
SELECT release_id, publish_digest, state, first_seen_at, published_at, heartbeat_at, retired_at
FROM release_lease
WHERE release_id = ?;

-- piko.query(name: MarkReleasePublished, command: exec)
UPDATE release_lease
SET state = 'published', published_at = ?, heartbeat_at = MAX(heartbeat_at, ?)
WHERE release_id = ?;

-- piko.query(name: HeartbeatRelease, command: exec)
UPDATE release_lease
SET heartbeat_at = ?
WHERE release_id = ? AND heartbeat_at < ?;

-- piko.query(name: ListExpiredReleases, command: many)
SELECT release_id
FROM release_lease
WHERE state = 'published' AND heartbeat_at < ? AND release_id <> ?;

-- piko.query(name: DeleteReleaseLease, command: exec)
DELETE FROM release_lease WHERE release_id = ?;

-- piko.query(name: DeleteStalePublishingLease, command: exec)
DELETE FROM release_lease
WHERE release_id = ? AND state = 'publishing' AND heartbeat_at < ?;
