CREATE TABLE users (
    id INT8 DEFAULT unique_rowid() PRIMARY KEY,
    name STRING NOT NULL,
    email STRING NOT NULL
);
