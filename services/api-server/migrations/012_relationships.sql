CREATE TABLE external_account_friend (
  id INTEGER PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES user(id),
  external_account_id TEXT NOT NULL,
  external_platform TEXT NOT NULL,
  friend_user_id TEXT REFERENCES user(id),
  friend_nickname TEXT NOT NULL DEFAULT '',
  friend_avatar TEXT,
  is_connected BOOLEAN DEFAULT 0,
  connected_at INTEGER,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_external_account_friend_user ON external_account_friend(user_id,id);

CREATE TABLE serendipity_match (
  id INTEGER PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES user(id),
  matched_user_id TEXT NOT NULL REFERENCES user(id),
  match_type TEXT NOT NULL,
  match_score REAL DEFAULT 0,
  matched_at INTEGER NOT NULL,
  expires_at INTEGER,
  status TEXT DEFAULT 'active',
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_serendipity_match_user ON serendipity_match(user_id,status);

CREATE TABLE star_tag_def (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  sku TEXT NOT NULL,
  part TEXT DEFAULT '',
  consumable_name TEXT NOT NULL,
  group_name TEXT DEFAULT '',
  batches TEXT DEFAULT '',
  can_link_with TEXT DEFAULT '',
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE star_tag_link (
  id INTEGER PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES user(id),
  tag_name TEXT NOT NULL,
  linked_item TEXT NOT NULL,
  linked_at INTEGER NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_star_tag_link_user ON star_tag_link(user_id,id);

CREATE TABLE server_assignment (
  id INTEGER PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES user(id),
  server_uri TEXT NOT NULL,
  move_ts INTEGER NOT NULL,
  signature TEXT NOT NULL,
  sig TEXT NOT NULL,
  ps_only BOOLEAN DEFAULT 0,
  platform_filter INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_server_assignment_user ON server_assignment(user_id,move_ts);

CREATE TABLE user_online_status (
  id INTEGER PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES user(id),
  friend_id TEXT NOT NULL REFERENCES user(id),
  is_online BOOLEAN DEFAULT 0,
  last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id,friend_id)
);

CREATE TABLE follow_relationship (
  id INTEGER PRIMARY KEY,
  follower_id TEXT NOT NULL REFERENCES user(id),
  following_id TEXT NOT NULL REFERENCES user(id),
  status TEXT DEFAULT 'active',
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(follower_id,following_id)
);
CREATE INDEX idx_follow_relationship_following ON follow_relationship(following_id,status);

CREATE TABLE boat_follow (
  user_id TEXT NOT NULL,
  creator TEXT NOT NULL,
  nickname TEXT NOT NULL DEFAULT '',
  last_modified TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (user_id,creator)
);
