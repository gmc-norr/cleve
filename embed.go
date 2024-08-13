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

//go:embed cleve_api.yaml
var cleve_api []byte

func GetAPIDoc() []byte {
	return cleve_api
}
