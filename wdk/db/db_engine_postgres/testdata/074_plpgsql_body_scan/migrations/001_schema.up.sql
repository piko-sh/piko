CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE FUNCTION count_events() RETURNS BIGINT
LANGUAGE plpgsql
AS $$
DECLARE
    result BIGINT;
BEGIN
    SELECT count(*) INTO result FROM events;
    RETURN result;
END;
$$;

CREATE FUNCTION log_event(event_name TEXT) RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO events (name) VALUES (event_name);
END;
$$;

CREATE FUNCTION delete_old_events() RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    DELETE FROM events WHERE occurred_at < now() - INTERVAL '30 days';
END;
$$;
