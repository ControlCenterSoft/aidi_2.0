package buildinfo

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

func New(version, commit, buildTime string) Info {
	if version == "" {
		version = "dev"
	}
	if commit == "" {
		commit = "unknown"
	}
	if buildTime == "" {
		buildTime = "unknown"
	}
	return Info{Version: version, Commit: commit, BuildTime: buildTime}
}
