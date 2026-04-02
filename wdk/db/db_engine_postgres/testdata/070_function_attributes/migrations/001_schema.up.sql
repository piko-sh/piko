CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price NUMERIC NOT NULL
);

CREATE FUNCTION double_price(val NUMERIC) RETURNS NUMERIC
LANGUAGE sql IMMUTABLE STRICT
AS $$ SELECT val * 2; $$;

CREATE FUNCTION get_current_price(product_id INTEGER) RETURNS NUMERIC
LANGUAGE sql STABLE
AS $$ SELECT price FROM products WHERE id = product_id; $$;

CREATE FUNCTION reset_prices() RETURNS VOID
LANGUAGE plpgsql VOLATILE
AS $$
BEGIN
    UPDATE products SET price = 0;
END;
$$;
