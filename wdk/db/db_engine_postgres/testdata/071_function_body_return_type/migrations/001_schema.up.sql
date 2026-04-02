CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_name TEXT NOT NULL,
    total NUMERIC NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE FUNCTION order_count() RETURNS BIGINT
LANGUAGE sql STABLE
AS $$ SELECT count(*) FROM orders; $$;

CREATE FUNCTION latest_customer() RETURNS TEXT
LANGUAGE sql STABLE
AS $$ SELECT customer_name FROM orders ORDER BY created_at DESC LIMIT 1; $$;
