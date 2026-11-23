package repository

const UserIDBySession Statement = `SELECT id FROM user WHERE session=?`
const UserIDByCredentialSession Statement = `SELECT id FROM user WHERE session=? AND credential_mode_issued=1`
