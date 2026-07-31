package main

import (
	"os"
	"reflect"
	"testing"
)

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		// Current scheme (year.release[.patch]).
		{"26.2", true},
		{"26.2.1", true},
		{"26.1.2", true},
		{"26.2-rc-2", true},

		// Legacy scheme.
		{"1.21.4", true},
		{"1.20", true},
		{"1.19.2", true},
		{"1.8.8", true},
		{"1.21.11-rc3", true},
		{"1.21.9-pre4", true},
		{"1.13-pre7", true},

		// Rejected.
		{"invalid", false},
		{"1.2.3.4", false},
		{"", false},
		{"1", false},
		{"1.21.4-", false},
		{"-1.21.4", false},
		{"1.21.4 ", false},
		{"../etc/passwd", false},
	}

	for _, test := range tests {
		if got := isValidVersion(test.version); got != test.valid {
			t.Errorf("isValidVersion(%q) = %v, want %v", test.version, got, test.valid)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1, v2   string
		expected int
	}{
		// Legacy ordering.
		{"1.21.4", "1.21.3", 1},
		{"1.21.3", "1.21.4", -1},
		{"1.21.4", "1.21.4", 0},
		{"1.21", "1.20.6", 1},
		{"1.20.6", "1.21", -1},
		{"1.20", "1.20.0", 0},
		{"1.21.11", "1.21.9", 1},

		// Current scheme, including against legacy versions.
		{"26.2", "26.1.2", 1},
		{"26.2.1", "26.2", 1},
		{"26.2", "1.21.11", 1},
		{"26.1.1", "26.1.2", -1},

		// A pre-release sorts below the release it leads up to.
		{"26.2", "26.2-rc-2", 1},
		{"1.21.11-rc3", "1.21.11", -1},
		{"1.21.11-rc3", "1.21.11-rc2", 1},
		{"1.21.11-rc3", "1.21.11-rc10", -1},
		{"1.21.9-pre4", "1.21.9-rc1", -1},
		{"26.2-rc-2", "26.2-rc-10", -1},
		{"1.21.11-rc3", "1.21.11-rc3", 0},
	}

	for _, test := range tests {
		if got := compareVersions(test.v1, test.v2); got != test.expected {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", test.v1, test.v2, got, test.expected)
		}
	}
}

func TestGetVersionsFromJars(t *testing.T) {
	t.Chdir(t.TempDir())

	jars := []string{
		"paper-1.21.4.jar",
		"paper-26.2.jar",
		"paper-26.2-rc-2.jar",
		"paper-1.8.8.jar",
		"paper-not-a-version.jar", // ignored: not a version
		"spigot-1.21.4.jar",       // ignored: not a Paper jar
	}
	for _, name := range jars {
		if err := os.WriteFile(name, nil, 0644); err != nil {
			t.Fatal(err)
		}
	}

	want := []string{"26.2", "26.2-rc-2", "1.21.4", "1.8.8"}
	if got := getVersionsFromJars(); !reflect.DeepEqual(got, want) {
		t.Errorf("getVersionsFromJars() = %v, want %v", got, want)
	}
}
