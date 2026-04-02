CREATE TABLE audit_log (
    id SERIAL PRIMARY KEY,
    action TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE OR REPLACE FUNCTION log_action(action_name TEXT) RETURNS VOID AS $$
DECLARE
    counter INTEGER := 0;
BEGIN
    INSERT INTO audit_log (action) VALUES (action_name);
    LOOP
        counter := counter + 1;
        EXIT WHEN counter > 10;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_recent_count() RETURNS BIGINT AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM audit_log WHERE created_at > now() - INTERVAL '1 hour');
END;
$$ LANGUAGE plpgsql;
