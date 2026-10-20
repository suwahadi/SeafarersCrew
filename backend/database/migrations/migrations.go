// Package migrations embeds the golang-migrate SQL files.
package migrations

import "embed"

// Files holds every *.sql migration.
//
//go:embed *.sql
var Files embed.FS
