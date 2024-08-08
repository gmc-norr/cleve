package cleve

import (
	"embed"
	"io/fs"
)

//go:embed templates assets
var assets embed.FS

func GetTemplateFS() (fs.FS, error) {
	return fs.Sub(assets, "templates")
}

func GetAssetFS() (fs.FS, error) {
	return fs.Sub(assets, "assets")
}
