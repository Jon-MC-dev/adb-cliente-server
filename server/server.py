import eventlet
eventlet.monkey_patch()

from flask import Flask
from flask_socketio import SocketIO

app = Flask(__name__, static_folder='../web', static_url_path='')
app.config['SECRET_KEY'] = 'secret!'
socketio = SocketIO(app, cors_allowed_origins="*")

@socketio.on('connect')
def handle_connect():
    print('Web client connected')

@socketio.on('output_from_client')
def handle_output(data):
    # Forward ADB output to web clients
    socketio.emit('output', data)

@socketio.on('command_from_web')
def handle_command(data):
    # Forward command to ADB client
    socketio.emit('execute_command', data)

@app.route('/')
def index():
    return app.send_static_file('index.html')

if __name__ == '__main__':
    socketio.run(app, host='0.0.0.0', port=5001)
