package migrations

import _ "embed"

// Initial is the original schema v1 SQL, retained without rewriting.
//
//go:embed 001_initial.sql
var Initial string

// Profile is the original schema v2 DDL. Version update remains transactional.
//
//go:embed 002_profile.sql
var Profile string

//go:embed 003_social.sql
var Social string

//go:embed 004_constellations.sql
var Constellations string

//go:embed 005_player_state.sql
var PlayerState string

//go:embed 006_invites.sql
var Invites string
