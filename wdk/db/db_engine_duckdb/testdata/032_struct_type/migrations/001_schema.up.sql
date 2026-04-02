CREATE TABLE sensors (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL,
    location STRUCT(latitude DOUBLE, longitude DOUBLE, label VARCHAR) NOT NULL
);
