CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  email text NOT NULL,
  active boolean NOT NULL DEFAULT true
);

CREATE VIEW active_users AS SELECT id, name, email FROM users WHERE active = true;
