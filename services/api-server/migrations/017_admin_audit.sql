CREATE TABLE admin_audit (
  id INTEGER PRIMARY KEY,
  admin_ip TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  user_id TEXT NOT NULL DEFAULT '',
  friend_id TEXT NOT NULL DEFAULT '',
  detail TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_admin_audit_user ON admin_audit(user_id,created_at);
