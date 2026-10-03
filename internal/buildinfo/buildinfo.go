package buildinfo

import (
	"encoding/json"
	"io"
)

type Info struct {
	Version    string `json:"version"`
	Commit     string `json:"commit"`
	BuildTime  string `json:"build_time"`
	Provenance string `json:"provenance"`
}

func New(version, commit, buildTime, provenance string) Info {
	if version == "" {
		version = "dev"
	}
	if commit == "" {
		commit = "unknown"
	}
	if buildTime == "" {
		buildTime = "unknown"
	}
	if provenance == "" {
		provenance = "unknown"
	}
	return Info{
		Version:    version,
		Commit:     commit,
		BuildTime:  buildTime,
		Provenance: provenance,
	}
}

func VersionRequested(args []string) bool {
	return len(args) == 2 && (args[1] == "--version" || args[1] == "version")
}

func WriteJSON(w io.Writer, info Info) error {
	return json.NewEncoder(w).Encode(info)
}
