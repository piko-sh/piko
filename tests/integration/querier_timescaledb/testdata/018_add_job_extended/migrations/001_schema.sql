CREATE PROCEDURE my_proc(job_id INTEGER, config JSONB)
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE NOTICE 'my_proc invoked with job_id=%', job_id;
END;
$$;
