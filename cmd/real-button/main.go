package main

import (
	_ "embed"
	"strings"

	"github.com/jphastings/real-button/cmd/real-button/cmd"
)

//go:embed VERSION
var versionFile string

func main() {
	cmd.Execute(strings.TrimSpace(versionFile))
}
