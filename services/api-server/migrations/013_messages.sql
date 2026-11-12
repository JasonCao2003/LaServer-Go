CREATE TABLE message (
  id TEXT NOT NULL,
  user_id TEXT NOT NULL REFERENCES user(id),
  received_at INTEGER,
  created_at INTEGER DEFAULT 0,
  updated_at INTEGER DEFAULT 0,
  PRIMARY KEY (id,user_id)
);
CREATE INDEX idx_message_user ON message(user_id,received_at);

CREATE TABLE free_gift (
  id INTEGER PRIMARY KEY,
  sender_id TEXT NOT NULL REFERENCES user(id),
  receiver_id TEXT NOT NULL REFERENCES user(id),
  gift_type TEXT NOT NULL,
  gift_id TEXT NOT NULL,
  sent_at INTEGER NOT NULL,
  received BOOLEAN DEFAULT 0,
  received_at INTEGER,
  expires_at INTEGER,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_free_gift_sender ON free_gift(sender_id,sent_at);
CREATE INDEX idx_free_gift_receiver ON free_gift(receiver_id,received,sent_at);
