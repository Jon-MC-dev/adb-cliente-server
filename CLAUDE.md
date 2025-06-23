# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is an ADB (Android Debug Bridge) remote shell system that provides web-based interaction with Android devices through a WebSocket-based architecture. The system consists of three main components:

1. **Server** (`server/server.py`) - Flask-SocketIO server that acts as a WebSocket relay
2. **Client** (`client/client.py`) - Python client that connects to ADB shell and forwards I/O
3. **Web Interface** (`web/`) - HTML/JavaScript frontend for command interaction

## Architecture

The system uses a relay architecture where:
- The Python client spawns an `adb shell` subprocess and connects to the server via SocketIO
- The Flask server relays messages between web clients and the ADB client
- Web clients connect directly to the Flask server and send commands through WebSocket events

**Key Event Flow:**
- Web → Server: `command_from_web` 
- Server → Client: `execute_command`
- Client → Server: `output_from_client`
- Server → Web: `output`

## Development Commands

### Server Setup
```bash
cd server
pip install -r requirements.txt
python server.py
```
Server runs on `http://0.0.0.0:5001`

### Client Setup  
```bash
cd client
pip install -r requirements.txt
python client.py http://localhost:5001
```

### Web Interface
Open `web/index.html` in browser (served by Flask server at root path)

## Dependencies

**Server:** flask, flask-socketio, eventlet
**Client:** python-socketio

## Prerequisites

- ADB installed and available in PATH
- Python 3.7+
- Android device connected and accessible via `adb shell`

## Configuration Notes

- Server URL in client defaults to `http://localhost:5001`
- Client hardcodes `adb shell` command - assumes single device or default device selection
- CORS enabled for all origins in Flask-SocketIO configuration
- Web interface uses CDN-hosted Socket.IO client library