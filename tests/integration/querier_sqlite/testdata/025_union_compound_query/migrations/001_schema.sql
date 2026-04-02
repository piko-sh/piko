CREATE TABLE inbox (
    id INTEGER PRIMARY KEY,
    subject TEXT NOT NULL,
    received_date TEXT NOT NULL
);

CREATE TABLE sent (
    id INTEGER PRIMARY KEY,
    subject TEXT NOT NULL,
    sent_date TEXT NOT NULL
);
