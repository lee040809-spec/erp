//go:build production_embed

package config

import _ "embed"

// Generated privately by cmd/package-linux; never commit this resource.
//
//go:embed production_embedded.env
var embeddedProduction string
