package main

import (
	"fmt"
	"io"
	"strings"
)

type command int

const (
	cmdRun command = iota
	cmdTest
	cmdVersion
	cmdVersions
	cmdHelp
)

type options struct {
	cmd       command
	temp      bool
	offline   bool
	mcVersion string
}

// parseArgs parses the command line. It has no side effects so it can be
// tested directly; main dispatches on the returned command.
func parseArgs(args []string) (options, error) {
	opts := options{cmd: cmdRun}

	for _, arg := range args {
		switch arg {
		case "test":
			opts.cmd = cmdTest
		case "versions":
			opts.cmd = cmdVersions
		case "version", "--version", "-v":
			opts.cmd = cmdVersion
		case "help", "--help", "-h":
			opts.cmd = cmdHelp
		case "--temp", "-t":
			opts.temp = true
		case "--offline", "--offlinemode", "-o":
			opts.offline = true
		default:
			if strings.HasPrefix(arg, "-") {
				return options{}, fmt.Errorf("unknown flag: %s", arg)
			}
			if opts.mcVersion != "" {
				return options{}, fmt.Errorf("unexpected argument: %s (a version is already set to %s)", arg, opts.mcVersion)
			}
			opts.mcVersion = arg
		}
	}

	return opts, nil
}

func printUsage(out io.Writer) {
	fmt.Fprint(out, `mc - quickly start Paper Minecraft servers

Usage:
  mc [flags] [version]

Commands:
  mc <version>        Start the given version, downloading it if needed
  mc                  Start using mc.yml, picking interactively if several
                      versions are already downloaded
  mc versions         List the versions PaperMC currently publishes
  mc test             Check that Java is installed and usable
  mc help             Show this help

Flags:
  -t, --temp          Run in a temporary directory, cleaned up on exit
  -o, --offline       Set online-mode=false in server.properties
      --offlinemode   Alias for --offline
  -v, --version       Print the mc version and exit
  -h, --help          Show this help

Versions may use the current scheme (26.2, 26.2.1), the legacy scheme
(1.21.4, 1.8.8), or a pre-release (1.21.11-rc3, 26.2-rc-2).

Examples:
  mc 26.2
  mc --temp --offline 1.21.4
`)
}
