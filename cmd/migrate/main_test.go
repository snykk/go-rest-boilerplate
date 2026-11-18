package main

import (
	"strings"
	"testing"
)

// run refuses a command it does not know, or the wrong number of arguments, with the usage before it reads
// the configuration; a command it knows reads the configuration first.
func TestRun(t *testing.T) {
	tCases := []struct {
		name    string
		args    []string
		wantErr string // text the error must hold
	}{
		{name: "no command is refused with the usage", wantErr: usage},
		{
			name:    "an unknown command is refused with the usage before the configuration is read",
			args:    []string{"sideways"},
			wantErr: usage,
		},
		{name: "create without a name is refused with the usage", args: []string{"create"}, wantErr: usage},
		{name: "extra arguments are refused with the usage", args: []string{"up", "extra"}, wantErr: usage},
		{name: "a known command reads the configuration first", args: []string{"up"}, wantErr: "DATABASE_URL"},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("DATABASE_URL", "")

			err := run(tc.args)

			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("run(%q) = %v, want an error holding %q", tc.args, err, tc.wantErr)
			}
		})
	}
}
