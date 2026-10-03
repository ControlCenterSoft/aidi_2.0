package main

import (
	"fmt"
	"os"

	"github.com/ControlCenterSoft/aidi_2.0/internal/buildinfo"
)

var (
	version    = "dev"
	commit     = "unknown"
	buildTime  = "unknown"
	provenance = "unknown"
)

func main() {
	if !buildinfo.VersionRequested(os.Args) {
		return
	}
	if err := buildinfo.WriteJSON(os.Stdout, buildinfo.New(version, commit, buildTime, provenance)); err != nil {
		fmt.Fprintln(os.Stderr, "write build metadata:", err)
		os.Exit(1)
	}
}
