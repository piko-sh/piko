CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    title STRING NOT NULL,
    content STRING,
    checksum BYTES
);
