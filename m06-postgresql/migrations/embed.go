// Package migrations embeds the goose SQL migration files so they ship inside
// the binary — no separate files to deploy.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
