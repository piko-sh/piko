-- piko.query(name: ScheduleJob, command: one)
WITH scheduled AS (
    SELECT add_job(
        'my_proc',
        INTERVAL '1 day',
        config         => '{}'::jsonb,
        fixed_schedule => true,
        timezone       => 'UTC'
    ) AS raw_job_id
)
SELECT raw_job_id::integer AS job_id FROM scheduled;

-- piko.query(name: DeleteJob, command: exec)
-- $1 as piko.param(jobID) int
SELECT delete_job($1);
