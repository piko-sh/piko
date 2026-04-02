CREATE SEQUENCE order_number_seq INCREMENT BY 1 START WITH 1000;

CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    order_number INTEGER NOT NULL DEFAULT nextval('order_number_seq')
);
