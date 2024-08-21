package main

import (
	_ "embed"
	"strings"
)

//go:generate sh -c "git describe --tags > version.txt || printf '' > version.txt"
//go:embed version.txt
var Version string
var LastRelease string = "v0.1.0" // x-release-please-version

func GetVersion() string {
	v := strings.TrimSpace(Version)
	if v == "" {
		return LastRelease
	}
	return v
}
