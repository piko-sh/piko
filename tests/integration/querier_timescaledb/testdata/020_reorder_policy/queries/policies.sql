-- piko.query(name: ScheduleReorderPolicy, command: one)
WITH scheduled AS (
    SELECT add_reorder_policy('readings', 'readings_ts_device_idx') AS raw_job_id
)
SELECT raw_job_id::integer AS job_id FROM scheduled;

-- piko.query(name: RemoveReorderPolicy, command: exec)
SELECT remove_reorder_policy('readings', true);
