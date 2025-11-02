package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetServerPropertyCreateAndUpdate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mc-props-test-*")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	_ = os.Chdir(tmpDir)

	// Create new file with a key
	if err := setServerProperty("online-mode", "false"); err != nil {
		t.Fatalf("setServerProperty create: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(tmpDir, "server.properties"))
	if err != nil {
		t.Fatalf("read server.properties: %v", err)
	}
	if !strings.Contains(string(b), "online-mode=false\n") {
		t.Fatalf("expected online-mode=false, got: %q", string(b))
	}

	// Update existing key
	if err := setServerProperty("online-mode", "true"); err != nil {
		t.Fatalf("setServerProperty update: %v", err)
	}
	b, _ = os.ReadFile(filepath.Join(tmpDir, "server.properties"))
	content := string(b)
	if !strings.Contains(content, "online-mode=true\n") {
		t.Fatalf("expected online-mode=true after update, got: %q", content)
	}
	if strings.Count(content, "online-mode=") != 1 {
		t.Fatalf("expected only one online-mode entry, got: %q", content)
	}
}
