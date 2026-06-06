-- piko.query(name: ClaimJob, command: one)
-- Plain UPDATE ... RETURNING * -- the star must expand to every column of
-- the UPDATE target so the emitter can declare the row type with the
-- correct fields.
UPDATE jobs
SET status = 'running', started_at = NOW()
WHERE id = $1
RETURNING *;

-- piko.query(name: ClaimNextJobs, command: many)
-- WITH cte AS (... FOR UPDATE SKIP LOCKED) UPDATE ... RETURNING *.
-- Reproduces the politepages scheduler claiming pattern: the outer
-- UPDATE references a CTE in its WHERE clause and returns every column
-- of the UPDATE target. RETURNING * must expand against `jobs`, not the
-- CTE.
WITH claimable AS (
    SELECT id FROM jobs
    WHERE status = 'pending'
    ORDER BY priority ASC
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE jobs
SET status = 'running', started_at = NOW()
WHERE id IN (SELECT id FROM claimable)
RETURNING *;

-- piko.query(name: InsertJob, command: one)
-- Sanity baseline: INSERT ... RETURNING * was already covered by test
-- 002 but having it next to the UPDATE variants here makes regressions
-- easier to spot.
INSERT INTO jobs (job_type) VALUES ($1) RETURNING *;

-- piko.query(name: DeleteCompletedJobs, command: many)
-- DELETE ... RETURNING * for completeness -- same expansion path as
-- UPDATE and INSERT.
DELETE FROM jobs WHERE status = 'completed' RETURNING *;

-- piko.query(name: ClaimQualifiedJob, command: one)
-- Schema-qualified target table with table.* -- schemas are no different
-- from any other target name; the star expansion logic must still find
-- the columns via the catalogue.
UPDATE jobs AS j SET status = 'running' WHERE j.id = $1 RETURNING j.*;

-- piko.query(name: ClaimWithSubquery, command: many)
-- No outer CTE -- just an inline subquery in the WHERE clause. If this
-- expands fine but ClaimNextJobs does not, the bug is specifically in
-- the analyseUpdate path when a CTE list precedes UPDATE.
UPDATE jobs
SET status = 'running'
WHERE id IN (SELECT id FROM jobs WHERE status = 'pending' LIMIT $1)
RETURNING *;

-- piko.query(name: ClaimWithSubqueryExplicitColumns, command: many)
-- Same shape but RETURNING explicit columns to isolate whether the
-- bug is in the * expansion or in the parser losing position before
-- RETURNING is reached.
UPDATE jobs
SET status = 'running'
WHERE id IN (SELECT id FROM jobs WHERE status = 'pending' LIMIT $1)
RETURNING id, job_type, status;

-- piko.query(name: ClaimNextJobsNoLockHint, command: many)
-- Same shape as ClaimNextJobs but without FOR UPDATE SKIP LOCKED, to
-- isolate whether the locking hint inside the CTE is what trips the
-- analyser.
WITH claimable AS (
    SELECT id FROM jobs
    WHERE status = 'pending'
    ORDER BY priority ASC
    LIMIT $1
)
UPDATE jobs
SET status = 'running', started_at = NOW()
WHERE id IN (SELECT id FROM claimable)
RETURNING *;

-- piko.query(name: InsertJobWithDefaults, command: one)
-- $1 as piko.param(priority, optional: true)
-- $2 as piko.param(status, optional: true)
-- $3 as piko.param(job_type)
-- INSERT VALUES list with COALESCE($n::type, default) pattern -- the
-- type should be inferred from the explicit ::int / ::text cast on
-- the parameter. The politepages CreateJobWithDeduplicationKey query
-- exercises the same shape; without proper type inference Piko emitted
-- `*any` fields and the downstream Go code failed to compile.
INSERT INTO jobs (job_type, priority, status)
VALUES (
    $3,
    COALESCE($1::int, 5),
    COALESCE($2::text, 'pending')
) RETURNING *;

-- piko.query(name: UpdateJobStatusWithCoalesce, command: exec)
-- $1 as piko.param(started_at, optional: true)
-- $2 as piko.param(attempts, optional: true)
-- $3 as piko.param(status)
-- $4 as piko.param(id)
-- SET clause with COALESCE wrappers around optional parameters -- each
-- inner parameter should pick up the explicit ::type cast and bind to
-- the LHS column so the emitted params struct has typed pointers like
-- *time.Time and *int32 rather than *any. Mirrors the politepages
-- scheduler UpdateJobStatus query shape.
UPDATE jobs
SET
    status = $3,
    started_at = COALESCE($1::timestamp, started_at),
    priority = COALESCE($2::int, priority)
WHERE id = $4;

-- piko.query(name: ClaimNextJobsWithCasts, command: many)
-- Reproducer for the cast-in-SET variant of the WHERE/RETURNING
-- termination bug: previously a SET assignment using a parameter cast
-- such as p::timestamp left the parser pointing at the wrong token,
-- and the WHERE walker overran RETURNING. This query mirrors the
-- politepages scheduler ClaimJobsForWorker exactly: a CTE prefix,
-- FOR UPDATE SKIP LOCKED, and multiple SET assignments using parameter
-- casts.
WITH claimable AS (
    SELECT id FROM jobs
    WHERE status = 'pending'
    ORDER BY priority ASC
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE jobs
SET status = 'running',
    started_at = $2::timestamp,
    priority = priority + 1
WHERE id IN (SELECT id FROM claimable)
RETURNING *;
