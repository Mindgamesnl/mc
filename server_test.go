package main

import (
	"os"
	"strings"
	"testing"
)

func TestSetServerPropertyCreateAndUpdate(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := setServerProperty("online-mode", "false"); err != nil {
		t.Fatalf("setServerProperty create: %v", err)
	}
	if got := readProps(t); !strings.Contains(got, "online-mode=false\n") {
		t.Fatalf("expected online-mode=false, got: %q", got)
	}

	if err := setServerProperty("online-mode", "true"); err != nil {
		t.Fatalf("setServerProperty update: %v", err)
	}
	got := readProps(t)
	if !strings.Contains(got, "online-mode=true\n") {
		t.Fatalf("expected online-mode=true after update, got: %q", got)
	}
	if n := strings.Count(got, "online-mode="); n != 1 {
		t.Fatalf("expected 1 online-mode entry, got %d: %q", n, got)
	}
}

func TestSetServerPropertyReplacesCommentedEntry(t *testing.T) {
	t.Chdir(t.TempDir())

	// Paper writes a fresh server.properties with commented defaults.
	if err := os.WriteFile("server.properties", []byte("#server-port=25565\nmotd=A Minecraft Server\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := setServerProperty("server-port", "25570"); err != nil {
		t.Fatalf("setServerProperty: %v", err)
	}

	got := readProps(t)
	if !strings.Contains(got, "server-port=25570\n") {
		t.Errorf("expected server-port=25570, got: %q", got)
	}
	if strings.Contains(got, "#server-port=") {
		t.Errorf("commented entry should have been replaced, got: %q", got)
	}
	if !strings.Contains(got, "motd=A Minecraft Server") {
		t.Errorf("unrelated properties should be preserved, got: %q", got)
	}
}

func TestAcceptEula(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := acceptEula(); err != nil {
		t.Fatalf("acceptEula() error = %v", err)
	}
	content, err := os.ReadFile("eula.txt")
	if err != nil {
		t.Fatalf("reading eula.txt: %v", err)
	}
	if !strings.Contains(string(content), "eula=true") {
		t.Errorf("eula.txt missing 'eula=true', got: %q", content)
	}
}

func readProps(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("server.properties")
	if err != nil {
		t.Fatalf("read server.properties: %v", err)
	}
	return string(b)
}
