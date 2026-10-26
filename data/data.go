package data

import "embed"

//go:embed minecraft
var FS embed.FS

//go:embed minecraft/registries.json
var RegistriesJSON []byte
