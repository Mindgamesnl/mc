package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

const configFile = "mc.yml"

const (
	defaultMemory = "2G"
	defaultPort   = 25565
)

// memoryPattern matches the JVM heap sizes accepted by -Xmx, e.g. 512M or 4G.
var memoryPattern = regexp.MustCompile(`^\d+[KkMmGg]?$`)

type Config struct {
	Version string `yaml:"version"`
	Memory  string `yaml:"memory"`
	Port    int    `yaml:"port"`
}

func defaultConfig() *Config {
	return &Config{Memory: defaultMemory, Port: defaultPort}
}

// loadConfig reads mc.yml from the working directory. A missing file is not
// an error; exists reports whether one was found.
func loadConfig() (config *Config, exists bool, err error) {
	data, err := os.ReadFile(configFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, false, fmt.Errorf("parsing %s: %w", configFile, err)
	}
	return &c, true, nil
}

func saveConfig(config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

// normalize fills in defaults for omitted fields and rejects values that
// would only fail later, once the JVM or the server has already started.
func (c *Config) normalize() error {
	if c.Memory == "" {
		c.Memory = defaultMemory
	}
	if c.Port == 0 {
		c.Port = defaultPort
	}

	if c.Version == "" {
		return fmt.Errorf("no version set in %s", configFile)
	}
	if !isValidVersion(c.Version) {
		return fmt.Errorf("invalid version in %s: %q", configFile, c.Version)
	}
	if !memoryPattern.MatchString(c.Memory) {
		return fmt.Errorf("invalid memory in %s: %q (expected e.g. 512M or 4G)", configFile, c.Memory)
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port in %s: %d (expected 1-65535)", configFile, c.Port)
	}
	return nil
}
