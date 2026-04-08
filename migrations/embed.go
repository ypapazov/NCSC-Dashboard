package migrations

import "embed"

// Files contains SQL migration scripts in lexical order by filename.
//
//go:embed *.sql
var Files embed.FS
