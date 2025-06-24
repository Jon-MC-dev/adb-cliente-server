# ADB Remote Client (Go)

Self-contained Go client for the ADB Remote system. No Python or runtime dependencies required.

## Building

### Quick Build (Current Platform)
```bash
go mod tidy
go build -o adb-remote-client main.go
```

### Multi-Platform Build
```bash
./build.sh
```

This creates binaries for:
- Windows (x64)
- Linux (x64, ARM64)  
- macOS (Intel, Apple Silicon)

## Usage

```bash
# Connect to local server
./adb-remote-client

# Connect to remote server
./adb-remote-client http://192.168.1.100:5001
./adb-remote-client https://myserver.com:5001
```

## Features

- **No Dependencies**: Single binary, no Python/Node.js required
- **Cross-Platform**: Windows, Linux, macOS
- **Terminal Modes**: 
  - `mode local` - Local system commands
  - `mode adb` - Android device shell
- **ADB Auto-Detection**: Checks local folder, then system PATH
- **Signal Handling**: Clean shutdown with Ctrl+C

## Commands

- `mode local` - Switch to local terminal
- `mode adb` - Switch to ADB shell  
- `help` - Show available commands
- `pwd` - Current directory (local mode)
- Any system command (local mode)
- Any ADB command (ADB mode)

## ADB Setup

Place ADB binaries in one of these locations:
- `adb/adb` (or `adb/adb.exe` on Windows)
- `platform-tools/adb` (or `platform-tools/adb.exe`)
- System PATH

## Binary Sizes

The compiled binaries are typically 8-12MB and include everything needed to run.