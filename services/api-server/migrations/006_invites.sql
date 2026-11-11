CREATE TABLE invite (
 id INTEGER PRIMARY KEY,
 inviter_user_id TEXT NOT NULL REFERENCES user(id),
 token_id TEXT NOT NULL UNIQUE,
 formatted_code TEXT,
 nickname TEXT NOT NULL,
 friend_id TEXT REFERENCES user(id),
 created INTEGER NOT NULL,
 updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
