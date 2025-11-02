package main

import "testing"

func TestParseArgsFlagsAndPositionals(t *testing.T) {
	temp, offline, pos, handled := parseArgs([]string{"--temp", "--offline", "1.21.4"})
	if handled {
		t.Fatalf("handled should be false")
	}
	if !temp || !offline {
		t.Fatalf("expected temp and offline to be true, got temp=%v offline=%v", temp, offline)
	}
	if len(pos) != 1 || pos[0] != "1.21.4" {
		t.Fatalf("unexpected positionals: %#v", pos)
	}
}

func TestParseArgsHandledVersionFlag(t *testing.T) {
	_, _, _, handled := parseArgs([]string{"--version"})
	if !handled {
		t.Fatalf("expected handled to be true when --version is passed")
	}
}
