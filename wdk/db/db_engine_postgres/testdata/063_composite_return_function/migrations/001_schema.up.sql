CREATE TYPE address_entry AS (
    street TEXT,
    city TEXT,
    postcode TEXT
);

CREATE TABLE contacts (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    addresses address_entry[] NOT NULL DEFAULT '{}'
);

CREATE FUNCTION expand_addresses(contact_id INTEGER)
RETURNS SETOF address_entry
LANGUAGE sql STABLE
AS $$
    SELECT unnest(addresses) FROM contacts WHERE id = contact_id;
$$;
