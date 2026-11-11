
CREATE TABLE laserver_schema(id INTEGER PRIMARY KEY CHECK(id=1),version INTEGER NOT NULL);
INSERT INTO laserver_schema VALUES(1,1);
CREATE TABLE user (
 id TEXT PRIMARY KEY NOT NULL, device TEXT DEFAULT '', device_identifier TEXT,
 session TEXT UNIQUE, recovery_token TEXT, version INTEGER, platform TEXT DEFAULT '',
 is_banned INTEGER DEFAULT 0,ban_duration INTEGER DEFAULT 0,ban_start_time INTEGER DEFAULT 0,ban_end_time INTEGER DEFAULT 0,
 is_authorized INTEGER DEFAULT 0,ip_address TEXT DEFAULT '0.0.0.0',created_at INTEGER,remark TEXT DEFAULT '',
 device_name TEXT DEFAULT '',device_type TEXT DEFAULT '',device_model TEXT DEFAULT '',device_os TEXT DEFAULT '',user_agent TEXT DEFAULT ''
);
CREATE TABLE user_currency (
 id INTEGER PRIMARY KEY, user_id TEXT NOT NULL UNIQUE REFERENCES user(id),
 candles INTEGER DEFAULT 0,heart INTEGER DEFAULT 0,heart_wax INTEGER DEFAULT 0,
 prestige INTEGER DEFAULT 0,prestige_vip INTEGER DEFAULT 0,prestige_wax INTEGER DEFAULT 0,
 season_candle INTEGER DEFAULT 0,season_wax INTEGER DEFAULT 0,season_wax_14 INTEGER DEFAULT 0,
 season_heart INTEGER DEFAULT 0,season_pass_token INTEGER DEFAULT 0,vip INTEGER DEFAULT 0,wax INTEGER DEFAULT 3,
 updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE user_data(user_id TEXT PRIMARY KEY NOT NULL REFERENCES user(id),data TEXT NOT NULL DEFAULT '',data_time INTEGER DEFAULT 0,data_version INTEGER DEFAULT 3,updated_at INTEGER DEFAULT 0);
CREATE TABLE user_rank(id INTEGER PRIMARY KEY,user_id TEXT NOT NULL UNIQUE REFERENCES user(id),social_rank REAL DEFAULT 0,is_shepherd BOOLEAN DEFAULT 0,is_beginner BOOLEAN DEFAULT 0,created_at DATETIME DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE user_checkpoint(user_id TEXT PRIMARY KEY NOT NULL REFERENCES user(id),checkpoints TEXT DEFAULT '[]');
CREATE TABLE season_pass_daily(user_id TEXT NOT NULL REFERENCES user(id),day TEXT NOT NULL,PRIMARY KEY(user_id,day));
CREATE TABLE version(id INTEGER PRIMARY KEY,version INTEGER);
