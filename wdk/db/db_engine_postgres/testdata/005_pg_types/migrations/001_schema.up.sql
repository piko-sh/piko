CREATE TABLE typed_data (
    id SERIAL PRIMARY KEY,
    data JSONB,
    tags TEXT[],
    metadata JSON,
    ip_addr INET,
    unique_id UUID NOT NULL DEFAULT gen_random_uuid(),
    price NUMERIC(10, 2),
    raw_data BYTEA
);
