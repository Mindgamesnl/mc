package main

import "testing"

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want options
	}{
		{
			name: "no arguments runs from config",
			args: nil,
			want: options{cmd: cmdRun},
		},
		{
			name: "flags and version",
			args: []string{"--temp", "--offline", "26.2"},
			want: options{cmd: cmdRun, temp: true, offline: true, mcVersion: "26.2"},
		},
		{
			name: "short flags",
			args: []string{"-t", "-o", "1.21.4"},
			want: options{cmd: cmdRun, temp: true, offline: true, mcVersion: "1.21.4"},
		},
		{
			name: "offlinemode alias",
			args: []string{"--offlinemode", "26.2.1"},
			want: options{cmd: cmdRun, offline: true, mcVersion: "26.2.1"},
		},
		{
			name: "version flag",
			args: []string{"--version"},
			want: options{cmd: cmdVersion},
		},
		{
			name: "versions subcommand",
			args: []string{"versions"},
			want: options{cmd: cmdVersions},
		},
		{
			name: "test subcommand",
			args: []string{"test"},
			want: options{cmd: cmdTest},
		},
		{
			name: "help flag",
			args: []string{"-h"},
			want: options{cmd: cmdHelp},
		},
		{
			name: "flags may follow the version",
			args: []string{"26.2", "--temp"},
			want: options{cmd: cmdRun, temp: true, mcVersion: "26.2"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseArgs(test.args)
			if err != nil {
				t.Fatalf("parseArgs(%v) error = %v", test.args, err)
			}
			if got != test.want {
				t.Errorf("parseArgs(%v) = %+v, want %+v", test.args, got, test.want)
			}
		})
	}
}

func TestParseArgsErrors(t *testing.T) {
	tests := [][]string{
		{"--nope"},         // unknown flag, previously ignored silently
		{"-x", "26.2"},     // unknown short flag
		{"26.2", "1.21.4"}, // two versions
	}

	for _, args := range tests {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("parseArgs(%v) error = nil, want an error", args)
		}
	}
}
