CREATE TYPE account_status AS ENUM ('pending', 'active', 'disabled');

CREATE TABLE accounts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  status account_status NOT NULL DEFAULT 'pending',
  name text NOT NULL
);

ALTER TABLE accounts ADD COLUMN email text NOT NULL;
