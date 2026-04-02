ALTER TABLE users RENAME COLUMN old_email TO email;
ALTER TABLE users ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();
