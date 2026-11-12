CREATE TABLE season_quest (user_id TEXT NOT NULL REFERENCES user(id), date TEXT NOT NULL, quest_id TEXT NOT NULL, stat_type TEXT NOT NULL, start_value REAL NOT NULL DEFAULT -1, activated INTEGER NOT NULL DEFAULT 0, PRIMARY KEY(user_id,date,quest_id));
CREATE TABLE season_quest_reward (user_id TEXT NOT NULL REFERENCES user(id), date TEXT NOT NULL, quest_id TEXT NOT NULL, rewarded INTEGER NOT NULL DEFAULT 0, PRIMARY KEY(user_id,date,quest_id));
CREATE TABLE season_marker (user_id TEXT NOT NULL REFERENCES user(id), name TEXT NOT NULL, created_at INTEGER NOT NULL, PRIMARY KEY(user_id,name));
CREATE TABLE player_quest (quest_id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES user(id), quest_name TEXT NOT NULL, start_time INTEGER NOT NULL, expiry_time INTEGER NOT NULL, status TEXT NOT NULL DEFAULT 'active');
CREATE INDEX player_quest_user_name ON player_quest(user_id,quest_name,start_time DESC);
CREATE TABLE quest_stat (quest_id TEXT NOT NULL REFERENCES player_quest(quest_id), stat_type TEXT NOT NULL, category TEXT NOT NULL, value INTEGER NOT NULL, PRIMARY KEY(quest_id,stat_type,category));
CREATE TABLE world_quest (user_id TEXT NOT NULL REFERENCES user(id), name TEXT NOT NULL, cooldown_over_time INTEGER NOT NULL, ever_completed INTEGER NOT NULL DEFAULT 1, PRIMARY KEY(user_id,name));
CREATE TABLE collected_pickup (user_id TEXT NOT NULL REFERENCES user(id), level_id TEXT NOT NULL, pickup_id TEXT NOT NULL, kind TEXT NOT NULL, date TEXT NOT NULL, PRIMARY KEY(user_id,level_id,pickup_id,kind,date));
