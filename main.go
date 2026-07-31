package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Version is set at build time via -ldflags "-X main.Version=...".
var Version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, errAborted) {
			os.Exit(130)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	opts, err := parseArgs(args)
	if err != nil {
		return fmt.Errorf("%w\nRun `mc --help` for usage", err)
	}

	switch opts.cmd {
	case cmdHelp:
		printUsage(os.Stdout)
		return nil
	case cmdVersion:
		fmt.Printf("mc version %s\n", Version)
		return nil
	case cmdTest:
		return testJavaSetup()
	case cmdVersions:
		return printAvailableVersions(os.Stdout)
	}

	if opts.temp {
		cleanup, err := useTempWorkdir()
		if err != nil {
			return fmt.Errorf("creating temp workdir: %w", err)
		}
		defer cleanup()
	}

	if err := validateJava(); err != nil {
		return err
	}

	config, err := resolveConfig(opts)
	if err != nil {
		return err
	}

	jarPath := fmt.Sprintf("paper-%s.jar", config.Version)
	if _, err := os.Stat(jarPath); errors.Is(err, os.ErrNotExist) {
		if err := downloadPaper(config.Version, jarPath); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if opts.offline {
		if err := setServerProperty("online-mode", "false"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to set online-mode=false: %v\n", err)
		}
	}
	if err := setServerProperty("server-port", strconv.Itoa(config.Port)); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to set server-port: %v\n", err)
	}

	fmt.Printf("Starting Minecraft server %s...\n", config.Version)
	return runServer(jarPath, config.Memory)
}

// resolveConfig decides which version to run, creating or updating mc.yml as
// needed. An explicit version argument wins; otherwise an existing config is
// used, with a picker when several versions are already downloaded.
func resolveConfig(opts options) (*Config, error) {
	config, exists, err := loadConfig()
	if err != nil {
		return nil, err
	}

	version := opts.mcVersion
	if !exists && version == "" {
		if version, err = promptForVersion(); err != nil {
			return nil, err
		}
	}

	if version != "" {
		if !isValidVersion(version) {
			return nil, fmt.Errorf("invalid version format: %q (expected e.g. 26.2, 26.2.1 or 1.21.4)", version)
		}
		if config == nil {
			config = defaultConfig()
		}
		// Only the version changes, so hand-tuned memory and port settings in
		// an existing mc.yml survive `mc <version>`.
		config.Version = version
		if err := config.normalize(); err != nil {
			return nil, err
		}
		if err := saveConfig(config); err != nil {
			return nil, fmt.Errorf("saving %s: %w", configFile, err)
		}
		return config, nil
	}

	if err := config.normalize(); err != nil {
		return nil, err
	}

	if versions := getVersionsFromJars(); len(versions) > 1 {
		selected, err := selectVersion(versions)
		if err != nil {
			return nil, err
		}
		config.Version = selected
	}
	return config, nil
}

func promptForVersion() (string, error) {
	fmt.Printf("No %s found. Enter Minecraft version (e.g., 26.2 or 1.21.4): ", configFile)

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", errAborted
	}
	version := strings.TrimSpace(scanner.Text())
	if version == "" {
		return "", errors.New("no version specified")
	}
	return version, nil
}

// printAvailableVersions lists what Paper currently publishes, grouped by
// version family and newest first.
func printAvailableVersions(out io.Writer) error {
	families, byFamily, err := availableVersions()
	if err != nil {
		return err
	}

	fmt.Fprintln(out, "Available Paper versions:")
	for _, family := range families {
		fmt.Fprintf(out, "  %-6s %s\n", family, strings.Join(byFamily[family], ", "))
	}
	return nil
}

// useTempWorkdir creates a temporary directory, chdirs into it, and returns a
// cleanup function that returns to the original directory and removes it.
func useTempWorkdir() (func(), error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// os.MkdirTemp with an empty dir honours TMPDIR and works on Windows,
	// unlike a hardcoded /tmp.
	tmpDir, err := os.MkdirTemp("", "mc-*")
	if err != nil {
		return nil, err
	}
	if err := os.Chdir(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}

	fmt.Printf("Using temporary workdir: %s\n", tmpDir)
	return func() {
		_ = os.Chdir(cwd)
		_ = os.RemoveAll(tmpDir)
	}, nil
}
