CREATE TABLE profiles (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  role TEXT NOT NULL
);

CREATE TABLE user_profiles (
  user_id INTEGER NOT NULL,
  profile_id INTEGER NOT NULL,
  PRIMARY KEY (user_id, profile_id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (profile_id) REFERENCES profiles(id)
);
