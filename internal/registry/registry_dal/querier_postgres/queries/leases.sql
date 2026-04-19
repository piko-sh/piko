-- piko.query(name: ClaimRelease, command: one, optional: true)
INSERT INTO registry_release_lease (release_id, publish_digest, state, first_seen_at, heartbeat_at)
VALUES ($1, $2, 'publishing', $3, $4)
ON CONFLICT(release_id) DO NOTHING
RETURNING release_id;

-- piko.query(name: GetRelease, command: one, optional: true)
SELECT release_id, publish_digest, state, first_seen_at, published_at, heartbeat_at, retired_at
FROM registry_release_lease
WHERE release_id = $1;

-- piko.query(name: MarkReleasePublished, command: exec)
UPDATE registry_release_lease
SET state = 'published', published_at = $2, heartbeat_at = GREATEST(heartbeat_at, $3)
WHERE release_id = $1;

-- piko.query(name: HeartbeatRelease, command: exec)
UPDATE registry_release_lease
SET heartbeat_at = $2
WHERE release_id = $1 AND heartbeat_at < $3;

-- piko.query(name: ListExpiredReleases, command: many)
SELECT release_id
FROM registry_release_lease
WHERE state = 'published' AND heartbeat_at < $1 AND release_id <> $2;

-- piko.query(name: DeleteReleaseLease, command: exec)
DELETE FROM registry_release_lease WHERE release_id = $1;

-- piko.query(name: DeleteStalePublishingLease, command: exec)
DELETE FROM registry_release_lease
WHERE release_id = $1 AND state = 'publishing' AND heartbeat_at < $2;
