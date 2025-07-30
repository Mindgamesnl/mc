package main

import (
	"os"
	"testing"
)

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		{"1.21.4", true},
		{"1.20", true},
		{"1.19.2", true},
		{"1.8.8", true},
		{"invalid", false},
		{"1.2.3.4", false},
		{"", false},
		{"1", false},
	}

	for _, test := range tests {
		result := isValidVersion(test.version)
		if result != test.valid {
			t.Errorf("isValidVersion(%s) = %v, want %v", test.version, result, test.valid)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1, v2   string
		expected int
	}{
		{"1.21.4", "1.21.3", 1},
		{"1.21.3", "1.21.4", -1},
		{"1.21.4", "1.21.4", 0},
		{"1.21", "1.20.6", 1},
		{"1.20.6", "1.21", -1},
		{"1.20", "1.20.0", 0},
	}

	for _, test := range tests {
		result := compareVersions(test.v1, test.v2)
		if result != test.expected {
			t.Errorf("compareVersions(%s, %s) = %d, want %d", test.v1, test.v2, result, test.expected)
		}
	}
}

func TestExtractLatestBuild(t *testing.T) {
	t.Log("Testing build number extraction from JSON...")

	t.Log("Testing with valid builds JSON")
	jsonBody := `{"builds":[{"build":123},{"build":124},{"build":125}]}`
	result := extractLatestBuild(jsonBody)
	expected := "125"

	if result != expected {
		t.Errorf("extractLatestBuild() = %s, want %s", result, expected)
	} else {
		t.Logf("✓ Extracted latest build: %s", result)
	}

	t.Log("Testing with empty builds JSON")
	emptyBody := `{"builds":[]}`
	result = extractLatestBuild(emptyBody)
	if result != "" {
		t.Errorf("extractLatestBuild() with empty builds = %s, want empty string", result)
	} else {
		t.Log("✓ Empty builds handled correctly")
	}

	t.Log("Testing with malformed JSON")
	malformedBody := `{"builds":[{"build":"not-a-number"}]}`
	result = extractLatestBuild(malformedBody)
	if result != "" {
		t.Errorf("extractLatestBuild() with malformed JSON = %s, want empty string", result)
	} else {
		t.Log("✓ Malformed JSON handled correctly")
	}
}

func TestLoadConfigNotExists(t *testing.T) {
	t.Log("Testing config loading when file doesn't exist...")

	// Ensure no mc.yml exists in current directory
	originalExists := false
	if _, err := os.Stat("mc.yml"); err == nil {
		originalExists = true
		os.Rename("mc.yml", "mc.yml.backup")
		defer os.Rename("mc.yml.backup", "mc.yml")
	}

	config, exists, err := loadConfig()
	if err != nil {
		t.Errorf("loadConfig() error = %v, want nil", err)
	}
	if exists {
		t.Errorf("loadConfig() exists = %v, want false", exists)
	}
	if config != nil {
		t.Errorf("loadConfig() config = %v, want nil", config)
	}

	if !originalExists {
		t.Log("✓ Non-existent config file handled correctly")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	t.Log("Testing config save and load operations...")

	tmpFile := "test_mc.yml"
	defer os.Remove(tmpFile)

	originalFile := "mc.yml"
	defer func() {
		os.Remove("mc.yml")
		if _, err := os.Stat(originalFile); err == nil {
			os.Rename(tmpFile, "mc.yml")
		}
	}()

	if _, err := os.Stat("mc.yml"); err == nil {
		os.Rename("mc.yml", tmpFile)
		t.Log("Backed up existing mc.yml")
	}

	config := &Config{
		Version: "1.21.4",
		Memory:  "4G",
	}

	t.Logf("Saving config: Version=%s, Memory=%s", config.Version, config.Memory)

	err := saveConfig(config)
	if err != nil {
		t.Errorf("saveConfig() error = %v, want nil", err)
		return
	}
	t.Log("✓ Config saved successfully")

	loadedConfig, exists, err := loadConfig()
	if err != nil {
		t.Errorf("loadConfig() error = %v, want nil", err)
		return
	}
	if !exists {
		t.Errorf("loadConfig() exists = %v, want true", exists)
		return
	}

	t.Logf("Loaded config: Version=%s, Memory=%s", loadedConfig.Version, loadedConfig.Memory)

	if loadedConfig.Version != config.Version {
		t.Errorf("loadConfig() version = %s, want %s", loadedConfig.Version, config.Version)
	}
	if loadedConfig.Memory != config.Memory {
		t.Errorf("loadConfig() memory = %s, want %s", loadedConfig.Memory, config.Memory)
	}

	t.Log("✓ Config round-trip successful")
}

func TestAcceptEula(t *testing.T) {
	t.Log("Testing EULA acceptance...")

	// Clean up any existing eula.txt
	defer os.Remove("eula.txt")

	err := acceptEula()
	if err != nil {
		t.Errorf("acceptEula() error = %v, want nil", err)
		return
	}

	// Check if file was created
	content, err := os.ReadFile("eula.txt")
	if err != nil {
		t.Errorf("Failed to read eula.txt: %v", err)
		return
	}

	contentStr := string(content)
	if !contains(contentStr, "eula=true") {
		t.Errorf("eula.txt doesn't contain 'eula=true', got: %s", contentStr)
		return
	}

	t.Log("✓ EULA file created with correct content")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
