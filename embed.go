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
var Version string
var LastRelease string = "v0.4.2" // x-release-please-version

func GetVersion() string {
	v := strings.TrimSpace(Version)
	if v == "" {
		return LastRelease
	}
	return v
}
