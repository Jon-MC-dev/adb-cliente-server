# ADB Remote Shell via WebSocket

## Prerequisites
- ADB installed and available in PATH
- Python 3.7+
- `pip install -r requirements.txt` for server and client

## Setup

### 1. Server
```bash
cd server
pip install -r requirements.txt
python server.py
```

### 2. Client
```bash
cd client
pip install -r requirements.txt
python client.py http://<SERVER_IP>:5000
```

### 3. Web
Open `web/index.html` in your browser (served by the Python server).

