-- piko.migration(no_transaction: true)
CREATE INDEX CONCURRENTLY idx_users_email ON users (email);
