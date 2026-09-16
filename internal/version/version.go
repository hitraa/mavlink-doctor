package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

var (
	// Version is the semantic version of the application (e.g. v0.1.0).
	Version = "v0.1.0"
	// Build is the git describe metadata.
	Build = "dev"
	// Commit is the git commit hash.
	Commit = "none"
	// BuildDate is the RFC3339 build timestamp.
	BuildDate = "unknown"
	// GoVersion is the Go runtime version.
	GoVersion = runtime.Version()
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		if GoVersion == "unknown" && info.GoVersion != "" {
			GoVersion = info.GoVersion
		}
		if (Version == "v0.1.0" || Version == "dev" || Version == "0.1.0-dev") && info.Main.Version != "" && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if Commit == "none" || Commit == "unknown" {
					Commit = setting.Value
					if len(Commit) > 7 {
						Commit = Commit[:7]
					}
				}
			case "vcs.time":
				if BuildDate == "unknown" {
					BuildDate = setting.Value
				}
			case "vcs.modified":
				if setting.Value == "true" && Build != "" && Build != "v0.1.0-dirty" {
					Build += "-dirty"
				}
			}
		}
	}
}

// String returns formatted multi-line version string.
func String() string {
	return fmt.Sprintf("mavlink-doctor %s (commit: %s, build: %s, date: %s, runtime: %s)",
		Version, Commit, Build, BuildDate, GoVersion)
}

// Short returns tool name and version.
func Short() string {
	return fmt.Sprintf("mavlink-doctor %s", Version)
}
