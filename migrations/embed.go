// Package migrations exposes the append-only SQL migrations embedded in the
// FolioPath binary.
package migrations

import "embed"

// FS contains every root-level Goose migration.
//
//go:embed *.sql
var FS embed.FS
