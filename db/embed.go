// Package db embeds the SQL migrations so the binary carries its schema.
package db

import "embed"

// Migrations holds the goose migrations under migrations/.
//
//go:embed migrations/*.sql
var Migrations embed.FS
