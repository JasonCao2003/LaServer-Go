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

//go:embed 007_unlocks.sql
var Unlocks string

//go:embed 008_resources.sql
var Resources string

//go:embed 009_magic_status.sql
var MagicStatus string

//go:embed 010_commerce.sql
var Commerce string

//go:embed 011_quests.sql
var Quests string

//go:embed 012_relationships.sql
var Relationships string

//go:embed 013_messages.sql
var Messages string

//go:embed 014_geonotes.sql
var Geonotes string

//go:embed 015_social_feed.sql
var SocialFeed string
