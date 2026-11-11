// Package model contains persistence records matching the legacy SQLite schema.
// Nullable columns use pointers, including columns with application defaults.
// These records are not API DTOs: JSON serialization intentionally omits fields.
// SQL defaults, constraints and foreign keys are documented in the schema report;
// they are not enforced by these structs or their db tags.
package model
