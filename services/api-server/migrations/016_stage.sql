CREATE TABLE stage_state (
  stage_id TEXT NOT NULL,
  owner_id TEXT NOT NULL DEFAULT '',
  props TEXT NOT NULL DEFAULT '[]',
  sequence INTEGER NOT NULL DEFAULT 1,
  level_id TEXT NOT NULL DEFAULT '',
  user_id TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (stage_id,owner_id)
);
