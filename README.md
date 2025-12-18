# mc - Minecraft Server Utility

A simple two-character command to quickly start Paper Minecraft servers for development and testing.
Just solves the minor inconvenience of getting out of your flow/terminal to test something quickly.

![Demo](.github/demo.gif)

## What it does

- Downloads and starts Paper Minecraft servers
- Handles EULA acceptance automatically (by using this tool, you accept Minecraft's EULA)
- Manages multiple server versions in the same directory
- Transparent stdio proxying with robust Ctrl+C handling (graceful stop then escalate)

## Installation

**Linux:**
```bash
curl -sfL https://github.com/Mindgamesnl/mc/releases/latest/download/mc-linux-amd64 -o mc && chmod +x mc && sudo mv mc /usr/local/bin/
```

**macOS:**
```
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
mc 1.21.4

# Run with existing setup (picks version interactively if multiple exist)  
mc

# Check if Java is working
mc test

# Run in a temporary work directory under /tmp and clean it after exit
mc --temp 1.21.4

# Force offline mode (sets online-mode=false in server.properties before start)
mc --offline 1.21.4
# alias
mc --offlinemode 1.21.4
```

## Flags

- `--temp`: Run in a temporary directory under `/tmp`, cleaned up on exit.
- `--offline`, `--offlinemode`: Set `online-mode=false` in `server.properties` before starting.
- `--version`, `-v`: Print version and exit.
- `test`: Test Java setup and exit.

## How it works

1. Creates `mc.yml` config file with defaults (2GB RAM, port 25565)
2. Downloads the Paper server jar for your version
3. Creates `eula.txt` (accepting Minecraft's EULA)
4. Presets `server.properties` based on flags (e.g., `online-mode=false`, `server-port`)
5. Starts the server and proxies stdio; Ctrl+C sends `stop`, then escalates signals if needed

If you have multiple versions downloaded, `mc` shows a menu to pick which one to run.

## Configuration

Edit `mc.yml` to change settings:

```yaml
version: "1.21.4"
memory: "2G"      # RAM allocation
port: 25565       # Server port
```

## Building

```bash
git clone https://github.com/Mindgamesnl/mc.git
cd mc
go build -o mc
```

# Honourable mention
To _Dr. Duco LeKuuk_ for complaining about how annoying it is to create temp servers, despite literally having the tool installed
