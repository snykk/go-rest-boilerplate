package buildinfo

import (
	"runtime/debug"
	"testing"
)

// resolve prefers the values set at link time, then what the Go toolchain recorded, then "dev" and no commit.
func TestResolve(t *testing.T) {
	recorded := &debug.BuildInfo{
		Main:     debug.Module{Version: "v1.2.3"},
		Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123"}},
	}
	tCases := []struct {
		name          string
		linkedVersion string
		linkedCommit  string
		recorded      *debug.BuildInfo
		want          Info
	}{
		{
			name:          "values set at link time win",
			linkedVersion: "v2.0.0",
			linkedCommit:  "fff000",
			recorded:      recorded,
			want:          Info{Version: "v2.0.0", Commit: "fff000"},
		},
		{
			name:     "without link-time values the recorded module version and revision are used",
			recorded: recorded,
			want:     Info{Version: "v1.2.3", Commit: "abc123"},
		},
		{
			name:     "a development build without a revision is dev with no commit",
			recorded: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			want:     Info{Version: "dev"},
		},
		{name: "no build information at all is dev with no commit", want: Info{Version: "dev"}},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolve(tc.linkedVersion, tc.linkedCommit, tc.recorded); got != tc.want {
				t.Errorf("resolve = %+v, want %+v", got, tc.want)
			}
		})
	}
}
