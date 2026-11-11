CREATE TABLE survey (
 id INTEGER PRIMARY KEY, title TEXT NOT NULL, description TEXT,
 survey_type TEXT NOT NULL, start_date TEXT NOT NULL, end_date TEXT NOT NULL,
 is_active INTEGER NOT NULL DEFAULT 1, created_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE user_survey (
 id INTEGER PRIMARY KEY, user_id TEXT NOT NULL REFERENCES user(id),
 survey_id INTEGER NOT NULL REFERENCES survey(id), completed INTEGER NOT NULL DEFAULT 0,
 completed_at TEXT, created_at TEXT DEFAULT CURRENT_TIMESTAMP,
 UNIQUE(user_id,survey_id)
);
CREATE INDEX idx_survey_active_dates ON survey(is_active,start_date,end_date);
CREATE TABLE video_definition (
 id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE, duration INTEGER NOT NULL,
 location TEXT NOT NULL, subtitle_base TEXT DEFAULT '', has_translations INTEGER DEFAULT 0,
 created_at TEXT DEFAULT CURRENT_TIMESTAMP, updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE video_schedule (
 id INTEGER PRIMARY KEY, channel TEXT NOT NULL, event TEXT NOT NULL,
 preshow_seconds INTEGER DEFAULT 0, playlist TEXT NOT NULL DEFAULT '[]',
 loop INTEGER DEFAULT 0, gap_seconds INTEGER DEFAULT 0, postshow_seconds INTEGER DEFAULT 0,
 silence_please INTEGER DEFAULT 0,
 created_at TEXT DEFAULT CURRENT_TIMESTAMP, updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE trust_activity_def (
 id INTEGER PRIMARY KEY, activity_id TEXT NOT NULL UNIQUE, activity_name TEXT NOT NULL,
 description TEXT, activity_type TEXT NOT NULL, required_level INTEGER DEFAULT 1,
 reward_points INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1,
 created_at TEXT DEFAULT CURRENT_TIMESTAMP, updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE trust_group_def (
 id INTEGER PRIMARY KEY, group_id TEXT NOT NULL UNIQUE, group_name TEXT NOT NULL,
 description TEXT, min_trust_level INTEGER DEFAULT 0, max_members INTEGER DEFAULT 50,
 is_active INTEGER DEFAULT 1,
 created_at TEXT DEFAULT CURRENT_TIMESTAMP, updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE user_trust_group (
 id INTEGER PRIMARY KEY, user_id TEXT NOT NULL REFERENCES user(id),
 group_id TEXT NOT NULL REFERENCES trust_group_def(group_id), trust_level INTEGER DEFAULT 1,
 joined_at TEXT DEFAULT CURRENT_TIMESTAMP, is_active INTEGER DEFAULT 1,
 UNIQUE(user_id,group_id)
);
CREATE INDEX idx_trust_group_active ON user_trust_group(group_id,is_active);
CREATE TABLE infraction (
 id INTEGER PRIMARY KEY, user_id TEXT NOT NULL REFERENCES user(id),
 infraction_type TEXT NOT NULL, reason TEXT, severity TEXT DEFAULT 'low',
 status TEXT DEFAULT 'active', start_time TEXT DEFAULT CURRENT_TIMESTAMP,
 end_time TEXT, created_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_infraction_user_status ON infraction(user_id,status,end_time);
CREATE TABLE level_pickup (
 id INTEGER PRIMARY KEY, user_id TEXT NOT NULL REFERENCES user(id), level INTEGER NOT NULL,
 emitter_name TEXT NOT NULL, event_name TEXT NOT NULL,
 preactive_duration INTEGER, pool_name TEXT NOT NULL, pool_amount INTEGER,
 last_chunk_seq INTEGER, emit_interval INTEGER, currency_type TEXT, collected INTEGER,
 chunks_per_emit INTEGER, chunk_size_likelihood TEXT, chunk_size_amount TEXT,
 active_duration INTEGER, created_at TEXT DEFAULT CURRENT_TIMESTAMP,
 UNIQUE(user_id,level,emitter_name,event_name)
);
CREATE INDEX idx_level_pickup_user_level ON level_pickup(user_id,level);
