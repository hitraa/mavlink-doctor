package version

import "fmt"

var (
	// Version is the semantic version of the application (e.g. v0.1.0).
	Version = "0.1.0-dev"
	// Build is the git describe metadata.
	Build = "dev"
	// Commit is the git commit hash.
	Commit = "none"
	// BuildDate is the RFC3339 build timestamp.
	BuildDate = "unknown"
	// GoVersion is the Go runtime version.
	GoVersion = "unknown"
)

// String returns formatted multi-line version string.
func String() string {
	return fmt.Sprintf("mavlink-doctor %s (commit: %s, build: %s, date: %s, runtime: %s)",
		Version, Commit, Build, BuildDate, GoVersion)
}

// Short returns tool name and version.
func Short() string {
	return fmt.Sprintf("mavlink-doctor %s", Version)
}
