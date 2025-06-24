const socket = io();
const consoleDiv = document.getElementById('console');
const input = document.getElementById('input');

socket.on('connect', () => {
  console.log('Connected to server');
});

socket.on('output', data => {
  console.log('Received output:', data);
  const line = document.createElement('div');
  line.textContent = data.output;
  consoleDiv.appendChild(line);
  consoleDiv.scrollTop = consoleDiv.scrollHeight;
});

input.addEventListener('keydown', event => {
  console.log(event.key);
  
  if (event.key === 'Enter') {
    const cmd = input.value;
    console.log('Sending command:', cmd);
    socket.emit('command_from_web', { command: cmd });
    input.value = '';
  }
});
