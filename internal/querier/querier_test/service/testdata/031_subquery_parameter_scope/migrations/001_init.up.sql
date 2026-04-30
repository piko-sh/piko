CREATE TABLE profiles (
  id int4 PRIMARY KEY,
  name text NOT NULL,
  role text NOT NULL
);

CREATE TABLE user_profiles (
  user_id int4 NOT NULL,
  profile_id int4 NOT NULL
);
