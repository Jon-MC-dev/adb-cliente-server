import socketio
import subprocess
import threading
import sys
import os
import signal
import time


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

# Global variables
adb_process = None
current_mode = "local"  # "local" or "adb"
current_directory = os.getcwd()
running = True

def signal_handler(signum, frame):
    global running, adb_process
    print("\nShutting down...")
    running = False
    
    if adb_process:
        try:
            adb_process.terminate()
            adb_process.wait(timeout=5)
        except:
            adb_process.kill()
    
    if sio.connected:
        sio.disconnect()
    
    sys.exit(0)

# Register signal handlers
signal.signal(signal.SIGINT, signal_handler)
signal.signal(signal.SIGTERM, signal_handler)

def start_adb_shell():
    """Start ADB shell process"""
    global adb_process
    try:
        adb_process = subprocess.Popen(
            [adb_path, 'shell'],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            bufsize=0  # Unbuffered
        )
        print("ADB shell process started")
        return True
    except Exception as e:
        print(f"Error starting ADB shell: {e}")
        return False

def execute_local_command(cmd):
    """Execute command on local system"""
    global current_directory
    
    try:
        # Handle cd command specially
        if cmd.strip().startswith('cd '):
            path = cmd.strip()[3:].strip()
            if not path:
                path = os.path.expanduser('~')
            elif path.startswith('~'):
                path = os.path.expanduser(path)
            elif not os.path.isabs(path):
                path = os.path.join(current_directory, path)
            
            if os.path.isdir(path):
                current_directory = os.path.abspath(path)
                return f"Changed directory to: {current_directory}"
            else:
                return f"cd: {path}: No such file or directory"
        
        # Execute other commands
        result = subprocess.run(
            cmd,
            shell=True,
            capture_output=True,
            text=True,
            cwd=current_directory,
            timeout=30
        )
        
        output = ""
        if result.stdout:
            output += result.stdout
        if result.stderr:
            output += result.stderr
            
        return output if output.strip() else f"Command executed (exit code: {result.returncode})"
        
    except subprocess.TimeoutExpired:
        return "Command timed out (30s limit)"
    except Exception as e:
        return f"Error executing command: {e}"

def execute_adb_command(cmd):
    """Execute command on ADB shell"""
    global adb_process
    
    if not adb_process or adb_process.poll() is not None:
        if not start_adb_shell():
            return "Failed to start ADB shell"
        # Start reading ADB output
        threading.Thread(target=read_adb_output, daemon=True).start()
    
    try:
        adb_process.stdin.write(cmd + '\n')
        adb_process.stdin.flush()
        return None  # Output will come through read_adb_output
    except Exception as e:
        return f"Error executing ADB command: {e}"

def read_adb_output():
    """Read output from ADB process"""
    global running, adb_process
    
    try:
        while running and adb_process and adb_process.poll() is None:
            line = ""
            while running:
                char = adb_process.stdout.read(1)
                if not char:
                    break
                line += char
                if char == '\n' or char == '\r':
                    if line.strip() and sio.connected:
                        sio.emit('output_from_client', {'output': line.strip()})
                    line = ""
    except Exception as e:
        if running:
            print(f"Error reading ADB output: {e}")

def get_prompt():
    """Get current prompt"""
    if current_mode == "local":
        return f"local:{os.path.basename(current_directory)}$ "
    else:
        return "adb$ "

@sio.event
def connect():
    print(f'Connected to server at {SERVER_URL}')
    # Send initial prompt
    if sio.connected:
        welcome_msg = f"Multi-Terminal Remote Console\nMode: {current_mode}\nCommands: 'mode local', 'mode adb', 'help'\n{get_prompt()}"
        sio.emit('output_from_client', {'output': welcome_msg})

@sio.event
def disconnect():
    print('Disconnected from server')
    global running
    running = False

@sio.on('execute_command')
def on_command(data):
    global current_mode
    
    try:
        cmd = data.get('command', '').strip()
        print(f"Executing command: {cmd} (mode: {current_mode})")
        
        # Handle mode switching commands
        if cmd == 'mode local':
            current_mode = "local"
            response = f"Switched to local mode\n{get_prompt()}"
        elif cmd == 'mode adb':
            current_mode = "adb"
            response = f"Switched to ADB mode\n{get_prompt()}"
        elif cmd == 'help':
            response = """Available commands:
- mode local: Switch to local terminal mode
- mode adb: Switch to ADB shell mode  
- help: Show this help
- Any system command (in local mode)
- Any ADB shell command (in ADB mode)"""
        elif cmd == 'pwd' and current_mode == "local":
            response = current_directory
        else:
            # Execute command based on current mode
            if current_mode == "local":
                response = execute_local_command(cmd)
            else:
                response = execute_adb_command(cmd)
        
        # Send response for local commands (ADB commands send output asynchronously)
        if response is not None and sio.connected:
            sio.emit('output_from_client', {'output': response + '\n' + get_prompt()})
            
    except Exception as e:
        error_msg = f"Error executing command: {e}\n{get_prompt()}"
        if sio.connected:
            sio.emit('output_from_client', {'output': error_msg})

try:
    print(f"Connecting to {SERVER_URL}...")
    sio.connect(SERVER_URL)
    
    # Keep alive loop instead of sio.wait()
    while running and sio.connected:
        time.sleep(0.1)
        
except KeyboardInterrupt:
    signal_handler(signal.SIGINT, None)
except Exception as e:
    print(f"Connection error: {e}")
    signal_handler(signal.SIGTERM, None)
