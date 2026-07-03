package assets

import "embed"

// FS expõe a pasta de assets embutidos.
//go:embed *
var FS embed.FS
