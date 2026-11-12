CREATE TABLE geonote (
  id INTEGER PRIMARY KEY,
  geonote_id TEXT NOT NULL UNIQUE,
  pool_type TEXT NOT NULL DEFAULT 'SeasonMeditationShrineNormal',
  pool_name TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL,
  author_id TEXT NOT NULL REFERENCES user(id),
  author_name TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  edited_at INTEGER,
  level_id INTEGER NOT NULL DEFAULT 0,
  level_hash INTEGER NOT NULL DEFAULT 0,
  pos_x REAL NOT NULL DEFAULT 0,
  pos_y REAL NOT NULL DEFAULT 0,
  pos_z REAL NOT NULL DEFAULT 0,
  parent_guid INTEGER NOT NULL DEFAULT 0,
  parent_rel_pos TEXT NOT NULL DEFAULT '[0,0,0]',
  cam_dir TEXT NOT NULL DEFAULT '[0,0,0]',
  likes INTEGER NOT NULL DEFAULT 0,
  liked BOOLEAN NOT NULL DEFAULT 0,
  visible BOOLEAN NOT NULL DEFAULT 1,
  reported BOOLEAN NOT NULL DEFAULT 0,
  emoji_id INTEGER NOT NULL DEFAULT 0,
  bug BOOLEAN NOT NULL DEFAULT 0,
  server_time INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_geonote_level ON geonote(bug,level_id,created_at);
CREATE INDEX idx_geonote_author ON geonote(author_id,created_at);
