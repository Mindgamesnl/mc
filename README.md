# mc - Minecraft Server Utility

A simple two-character command to quickly start Paper Minecraft servers for development and testing.
Just solves the minor inconvenience of getting out of your flow/terminal to test something quickly.

![Demo](.github/demo.gif)

## What it does

- Downloads and starts Paper Minecraft servers
- Handles EULA acceptance automatically (by using this tool, you accept Minecraft's EULA)
- Manages multiple server versions in the same directory
- Verifies every download against the checksum PaperMC publishes
- Transparent stdio proxying with robust Ctrl+C handling (graceful stop then escalate)

## Installation

**Linux (x64):**
```bash
curl -sfL https://github.com/Mindgamesnl/mc/releases/latest/download/mc-linux-amd64 -o mc && chmod +x mc && sudo mv mc /usr/local/bin/
```

**Linux (ARM64):**
```bash
curl -sfL https://github.com/Mindgamesnl/mc/releases/latest/download/mc-linux-arm64 -o mc && chmod +x mc && sudo mv mc /usr/local/bin/
```

**macOS (Apple Silicon):**
```bash
curl -sfL https://github.com/Mindgamesnl/mc/releases/latest/download/mc-darwin-arm64 -o mc && chmod +x mc && sudo mv mc /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -sfL https://github.com/Mindgamesnl/mc/releases/latest/download/mc-darwin-amd64 -o mc && chmod +x mc && sudo mv mc /usr/local/bin/
```

**Windows PowerShell:**
```powershell
Invoke-WebRequest -Uri "https://github.com/Mindgamesnl/mc/releases/latest/download/mc-windows-amd64.exe" -OutFile "mc.exe"
```

**Manual:** Download from [releases](https://github.com/Mindgamesnl/mc/releases), make executable, and add to PATH.

## Usage

```bash
# Start a specific version (downloads if needed)
mc 26.2

# Run with existing setup (picks version interactively if multiple exist)
mc

# List every version PaperMC currently publishes
mc versions

# Check if Java is working
mc test

# Run in a temporary work directory, cleaned up on exit
mc --temp 26.2

# Force offline mode (sets online-mode=false in server.properties before start)
mc --offline 26.2
```

## Versions

Minecraft moved to a `year.release[.patch]` scheme in 2026, so both of these work,
alongside anything else PaperMC still publishes:

| Scheme | Examples |
| --- | --- |
| Current | `26.2`, `26.2.1`, `26.1.2` |
| Legacy | `1.21.4`, `1.20`, `1.8.8` |
| Pre-releases | `26.2-rc-2`, `1.21.11-rc3`, `1.21.9-pre4` |

`mc versions` lists what is available right now. When several versions are already
downloaded in a directory, plain `mc` shows a picker; they are ordered newest first,
with a pre-release sorting just below the release it leads up to.

## Flags

- `-t`, `--temp`: Run in a temporary directory, cleaned up on exit.
- `-o`, `--offline`, `--offlinemode`: Set `online-mode=false` in `server.properties`.
- `-v`, `--version`: Print the mc version and exit.
- `-h`, `--help`: Show usage.

Subcommands: `mc versions`, `mc test`, `mc help`.

## How it works

1. Creates `mc.yml` config file with defaults (2GB RAM, port 25565)
2. Asks the [PaperMC Fill API](https://fill.papermc.io/) for the latest build of your version
3. Downloads the server jar, verifies its SHA-256, and only then puts it in place
4. Creates `eula.txt` (accepting Minecraft's EULA)
5. Presets `server.properties` based on flags (e.g. `online-mode=false`, `server-port`)
6. Starts the server and proxies stdio; Ctrl+C sends `stop`, then escalates if needed

Shutdown is staged: the first Ctrl+C (or `SIGTERM`) sends the `stop` console command,
10 seconds later the process group is terminated, and 5 seconds after that it is killed.
A third interrupt kills it immediately.

## Configuration

Edit `mc.yml` to change settings:

```yaml
version: "26.2"
memory: "2G"      # RAM allocation
port: 25565       # Server port
```

Running `mc <version>` updates only the `version` field, so hand-tuned memory and port
settings are preserved.

## Building

```bash
git clone https://github.com/Mindgamesnl/mc.git
cd mc
make build      # or: go build -o mc
make test       # tests with the race detector
make lint       # gofmt + go vet
```

Requires Go 1.25 or newer.

# Honourable mention
To _Dr. Duco LeKuuk_ for complaining about how annoying it is to create temp servers, despite literally having the tool installed
