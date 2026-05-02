CREATE TABLE content_media_folders (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE content_media_folder_versions (
    id INTEGER PRIMARY KEY,
    media_folder_id INTEGER NOT NULL,
    status TEXT NOT NULL
);
