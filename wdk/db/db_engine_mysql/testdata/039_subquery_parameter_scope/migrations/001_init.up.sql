CREATE TABLE profiles (
    id INT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    role VARCHAR(50) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE user_profiles (
    user_id INT NOT NULL,
    profile_id INT NOT NULL,
    PRIMARY KEY (user_id, profile_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
