CREATE TABLE tasks (
  id INTEGER PRIMARY KEY,
  priority INTEGER NOT NULL,
  assigned_to TEXT,
  completed INTEGER NOT NULL DEFAULT 0
);
