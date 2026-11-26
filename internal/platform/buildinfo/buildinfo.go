// Package buildinfo reports the version and the commit a binary was built from.
package buildinfo

import "runtime/debug"

// Set at link time: go build -ldflags "-X <this package>.version=v1.2.3 -X <this package>.commit=abc123".
var (
	version string
	commit  string
)

// Info names the build of a binary.
type Info struct {
	// Version is the release version, such as v1.2.3, or "dev" for a development build.
	Version string
	// Commit is the VCS revision, or "" when it is unknown.
	Commit string
}

// Get returns the version and commit set at link time. Without them it returns what the Go toolchain recorded
// in the binary, and "dev" with no commit when that is missing too.
func Get() Info {
	recorded, _ := debug.ReadBuildInfo()
	return resolve(version, commit, recorded)
}

func resolve(linkedVersion, linkedCommit string, recorded *debug.BuildInfo) Info {
	info := Info{Version: linkedVersion, Commit: linkedCommit}
	if recorded != nil {
		if info.Version == "" && recorded.Main.Version != "(devel)" {
			info.Version = recorded.Main.Version
		}
		if info.Commit == "" {
			info.Commit = revision(recorded)
		}
	}

	if info.Version == "" {
		info.Version = "dev"
	}
	return info
}

func revision(recorded *debug.BuildInfo) string {
	for _, setting := range recorded.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}
	return ""
}
