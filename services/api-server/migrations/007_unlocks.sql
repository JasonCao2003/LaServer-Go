CREATE TABLE unlock_item (
 id INTEGER PRIMARY KEY,
 user_id TEXT NOT NULL REFERENCES user(id),
 name TEXT NOT NULL,
 unlock_type TEXT NOT NULL,
 ack BOOLEAN DEFAULT 0,
 created_at INTEGER NOT NULL,
 UNIQUE(user_id,name)
);
CREATE INDEX idx_unlock_item_user_created ON unlock_item(user_id,created_at DESC);
