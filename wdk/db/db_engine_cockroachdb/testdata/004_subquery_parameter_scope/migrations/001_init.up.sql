CREATE TABLE profiles (
  id INT4 PRIMARY KEY,
  name TEXT NOT NULL,
  role TEXT NOT NULL
);

CREATE TABLE user_profiles (
  user_id INT4 NOT NULL,
  profile_id INT4 NOT NULL,
  PRIMARY KEY (user_id, profile_id)
);
