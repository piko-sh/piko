CREATE TABLE measurements (
    id INTEGER PRIMARY KEY,
    value DOUBLE NOT NULL
);

CREATE MACRO double_value(x) AS x * 2;

CREATE MACRO add_label(v) AS 'result: ' || CAST(v AS VARCHAR);
