CREATE TABLE contacts (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL,
    address STRUCT(street VARCHAR, city VARCHAR, zip VARCHAR)
);
