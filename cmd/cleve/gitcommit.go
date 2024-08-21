package main

import _ "embed"

//go:generate sh -c "printf %s $(git describe --tags) > version.txt"
//go:embed version.txt
var Version string
