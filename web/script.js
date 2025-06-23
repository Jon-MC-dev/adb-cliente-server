const socket = io();
const consoleDiv = document.getElementById('console');
const input = document.getElementById('input');

socket.on('connect', () => {
  console.log('Connected to server');
});

socket.on('output', data => {
  const line = document.createElement('div');
  line.textContent = data.output;
  consoleDiv.appendChild(line);
  consoleDiv.scrollTop = consoleDiv.scrollHeight;
});

input.addEventListener('keydown', event => {
  if (event.key === 'Enter') {
    const cmd = input.value;
    socket.emit('command_from_web', { command: cmd });
    input.value = '';
  }
});
