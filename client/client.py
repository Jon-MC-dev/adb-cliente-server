import socketio
import subprocess
import threading
import sys
import os


SERVER_URL = sys.argv[1] if len(sys.argv) > 1 else "https://lsk6q09x-5001.usw3.devtunnels.ms"

sio = socketio.Client()

def find_adb():
    """Find ADB executable, checking local folder first, then PATH"""
    # Check local adb folder first
    local_adb_paths = [
        os.path.join(os.path.dirname(__file__), 'adb', 'adb'),
        os.path.join(os.path.dirname(__file__), 'adb', 'adb.exe'),
        os.path.join(os.path.dirname(__file__), 'platform-tools', 'adb'),
        os.path.join(os.path.dirname(__file__), 'platform-tools', 'adb.exe')
    ]
    
    for adb_path in local_adb_paths:
        if os.path.isfile(adb_path):
            try:
                subprocess.run([adb_path, 'version'], check=True, capture_output=True)
                print(f"Using local ADB: {adb_path}")
                return adb_path
            except (subprocess.CalledProcessError, FileNotFoundError):
                continue
    
    # Check system PATH
    try:
        subprocess.run(['adb', 'version'], check=True, capture_output=True)
        print("Using system ADB from PATH")
        return 'adb'
    except (subprocess.CalledProcessError, FileNotFoundError):
        return None

# Find ADB executable
adb_path = find_adb()
if not adb_path:
    print("Error: ADB not found.")
    print("Options:")
    print("1. Install ADB and add it to PATH")
    print("2. Place ADB binaries in 'adb/' or 'platform-tools/' subfolder")
    sys.exit(1)

# Start ADB shell process
try:
    process = subprocess.Popen(
        [adb_path, 'shell'],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True
    )
except Exception as e:
    print(f"Error starting ADB shell: {e}")
    sys.exit(1)

def read_output():
    try:
        for line in process.stdout:
            if sio.connected:
                sio.emit('output_from_client', {'output': line})
    except Exception as e:
        print(f"Error reading ADB output: {e}")

@sio.event
def connect():
    print(f'Connected to server at {SERVER_URL}')
    # Start reading output only after connection
    threading.Thread(target=read_output, daemon=True).start()

@sio.event
def disconnect():
    print('Disconnected from server')

@sio.on('execute_command')
def on_command(data):
    try:
        cmd = data.get('command', '')
        process.stdin.write(cmd + '\n')
        process.stdin.flush()
    except Exception as e:
        print(f"Error executing command: {e}")

try:
    print(f"Connecting to {SERVER_URL}...")
    sio.connect(SERVER_URL)
    sio.wait()
except Exception as e:
    print(f"Connection error: {e}")
    sys.exit(1)
