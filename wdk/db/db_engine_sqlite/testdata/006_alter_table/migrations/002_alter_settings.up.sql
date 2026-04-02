ALTER TABLE settings ADD COLUMN description TEXT;
ALTER TABLE settings RENAME COLUMN key TO setting_key;
