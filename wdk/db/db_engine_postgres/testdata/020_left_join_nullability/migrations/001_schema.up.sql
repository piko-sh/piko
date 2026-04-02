CREATE TABLE departments (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    dept_id INTEGER REFERENCES departments(id)
);
