CREATE TABLE social_feed_post (
  id INTEGER PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES user(id),
  social_feed_id TEXT NOT NULL UNIQUE,
  pool_type TEXT NOT NULL,
  pool_name TEXT NOT NULL,
  message TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL DEFAULT 0,
  expire_at INTEGER NOT NULL DEFAULT 0,
  likes_count INTEGER NOT NULL DEFAULT 0,
  comments_count INTEGER NOT NULL DEFAULT 0,
  report_count INTEGER NOT NULL DEFAULT 0,
  hide_count INTEGER NOT NULL DEFAULT 0,
  attempt_count INTEGER NOT NULL DEFAULT 0,
  succeed_count INTEGER NOT NULL DEFAULT 0,
  is_poisoned BOOLEAN NOT NULL DEFAULT 0
);
CREATE INDEX idx_social_feed_pool ON social_feed_post(pool_type,pool_name,expire_at,created_at);
CREATE INDEX idx_social_feed_user ON social_feed_post(user_id,created_at);

CREATE TABLE social_feed_like (
  id INTEGER PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES user(id),
  social_feed_id TEXT NOT NULL,
  created_at INTEGER NOT NULL DEFAULT 0,
  UNIQUE(user_id,social_feed_id)
);
CREATE INDEX idx_social_feed_like_feed ON social_feed_like(social_feed_id);

CREATE TABLE social_feed_extra (
  social_feed_id TEXT PRIMARY KEY,
  extra_data TEXT NOT NULL DEFAULT '{}',
  resource_id TEXT NOT NULL DEFAULT '',
  comments_enabled BOOLEAN NOT NULL DEFAULT 1,
  tags TEXT,
  private BOOLEAN NOT NULL DEFAULT 0,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_social_feed_extra_resource ON social_feed_extra(resource_id);
