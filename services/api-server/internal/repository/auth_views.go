package repository

const UserIDBySession Statement = `SELECT id FROM user WHERE session=?`
