package cleve

import (
	"embed"
	"io/fs"
	"strings"
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

//go:generate sh -c "git describe --tags > version.txt || printf '' > version.txt"
//go:embed version.txt
var version string
var lastRelease string = "v0.7.0" // x-release-please-version

func GetVersion() string {
	v := strings.TrimSpace(version)
	if v == "" {
		return lastRelease
	}
	return v
}
