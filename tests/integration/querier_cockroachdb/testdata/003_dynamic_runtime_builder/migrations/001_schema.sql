CREATE TABLE products (
    id INT8 DEFAULT unique_rowid() PRIMARY KEY,
    name STRING NOT NULL,
    price INT8 NOT NULL,
    in_stock BOOL NOT NULL
);
