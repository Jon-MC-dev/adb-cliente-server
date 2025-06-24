package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// Client configuration
type Config struct {
	ServerURL        string
	CurrentMode      string // "local" or "adb"
	CurrentDirectory string
	ADBPath          string
}

// Client represents the WebSocket client
type Client struct {
	config     *Config
	conn       *websocket.Conn
	adbProcess *exec.Cmd
	adbStdin   io.WriteCloser
	adbStdout  io.ReadCloser
	running    bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// Message structures for SocketIO-like protocol
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type CommandData struct {
	Command string `json:"command"`
}

type OutputData struct {
	Output string `json:"output"`
}

func main() {
	// Parse server URL from command line
	serverURL := "http://localhost:5001"
	if len(os.Args) > 1 {
		serverURL = os.Args[1]
	}

	// Find ADB executable
	adbPath := findADB()
	if adbPath == "" {
		fmt.Println("Warning: ADB not found. ADB mode will not be available.")
		fmt.Println("Options:")
		fmt.Println("1. Install ADB and add it to PATH")
		fmt.Println("2. Place ADB binaries in 'adb/' or 'platform-tools/' subfolder")
		fmt.Println("Continuing in local-only mode...")
	}

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current directory:", err)
	}

	// Create config
	config := &Config{
		ServerURL:        serverURL,
		CurrentMode:      "local",
		CurrentDirectory: currentDir,
		ADBPath:          adbPath,
	}

	// Create client
	client := NewClient(config)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		client.Stop()
		os.Exit(0)
	}()

	// Connect and run
	if err := client.Connect(); err != nil {
		log.Fatal("Failed to connect:", err)
	}

	client.Run()
}

func NewClient(config *Config) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		config:  config,
		running: true,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (c *Client) Connect() error {
	// Convert HTTP URL to WebSocket URL with SocketIO path
	u, err := url.Parse(c.config.ServerURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %v", err)
	}

	if u.Scheme == "http" {
		u.Scheme = "ws"
	} else if u.Scheme == "https" {
		u.Scheme = "wss"
	}

	u.Path = "/socket.io/"
	q := u.Query()
	q.Set("EIO", "4")
	q.Set("transport", "websocket")
	u.RawQuery = q.Encode()

	fmt.Printf("Connecting to %s...\n", u.String())

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("websocket connection failed: %v", err)
	}

	c.conn = conn
	fmt.Printf("Connected to server at %s\n", c.config.ServerURL)

	// Send initial connection messages
	c.sendMessage("40") // SocketIO connect message

	// Send welcome message
	time.Sleep(100 * time.Millisecond)
	welcome := "Multi-Terminal Remote Console (Go Client)\nMode: " + c.config.CurrentMode + "\nCommands: 'mode local'"
	if c.config.ADBPath != "" {
		welcome += ", 'mode adb'"
	}
	welcome += ", 'help'\n" + c.getPrompt()
	c.sendOutput(welcome)

	return nil
}

func (c *Client) Run() {
	// Start message handler
	go c.handleMessages()

	// Keep running until stopped
	for c.running {
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *Client) Stop() {
	c.running = false
	c.cancel()

	if c.adbProcess != nil {
		c.adbProcess.Process.Kill()
	}

	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) handleMessages() {
	for c.running {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if c.running {
					log.Printf("Error reading message: %v", err)
				}
				return
			}

			c.processMessage(string(message))
		}
	}
}

func (c *Client) processMessage(message string) {
	// Skip engine.io protocol messages
	if len(message) == 0 {
		return
	}

	// Handle different message types
	switch message[0] {
	case '0': // open
		fmt.Println("Connection opened")
	case '2': // ping
		c.sendMessage("3") // pong
	case '4': // socket.io message
		if len(message) > 1 {
			c.handleSocketIOMessage(message[1:])
		}
	}
}

func (c *Client) handleSocketIOMessage(data string) {
	// Parse SocketIO message
	if len(data) < 1 {
		return
	}

	msgType := data[0]
	payload := data[1:]

	switch msgType {
	case '0': // connect
		fmt.Println("SocketIO connected")
	case '2': // event
		c.handleEvent(payload)
	}
}

func (c *Client) handleEvent(payload string) {
	var eventData []interface{}
	if err := json.Unmarshal([]byte(payload), &eventData); err != nil {
		log.Printf("Failed to parse event: %v", err)
		return
	}

	if len(eventData) < 1 {
		return
	}

	eventName, ok := eventData[0].(string)
	if !ok {
		return
	}

	switch eventName {
	case "execute_command":
		if len(eventData) > 1 {
			if cmdData, ok := eventData[1].(map[string]interface{}); ok {
				if cmd, ok := cmdData["command"].(string); ok {
					c.executeCommand(cmd)
				}
			}
		}
	}
}

func (c *Client) executeCommand(cmd string) {
	cmd = strings.TrimSpace(cmd)
	fmt.Printf("Executing command: %s (mode: %s)\n", cmd, c.config.CurrentMode)

	var response string

	// Handle mode switching commands
	switch cmd {
	case "mode local":
		c.config.CurrentMode = "local"
		response = "Switched to local mode\n" + c.getPrompt()
	case "mode adb":
		if c.config.ADBPath == "" {
			response = "ADB not available. Please install ADB first.\n" + c.getPrompt()
		} else {
			c.config.CurrentMode = "adb"
			response = "Switched to ADB mode\n" + c.getPrompt()
		}
	case "help":
		help := `Available commands:
- mode local: Switch to local terminal mode`
		if c.config.ADBPath != "" {
			help += `
- mode adb: Switch to ADB shell mode`
		}
		help += `
- help: Show this help
- Any system command (in local mode)`
		if c.config.ADBPath != "" {
			help += `
- Any ADB shell command (in ADB mode)`
		}
		response = help
	case "pwd":
		if c.config.CurrentMode == "local" {
			response = c.config.CurrentDirectory
		} else {
			c.executeADBCommand(cmd)
			return
		}
	default:
		// Execute command based on current mode
		if c.config.CurrentMode == "local" {
			response = c.executeLocalCommand(cmd)
		} else if c.config.ADBPath != "" {
			c.executeADBCommand(cmd)
			return // ADB commands send output asynchronously
		} else {
			response = "ADB mode not available\n" + c.getPrompt()
		}
	}

	// Send response for local commands
	if response != "" {
		c.sendOutput(response + "\n" + c.getPrompt())
	}
}

func (c *Client) executeLocalCommand(cmd string) string {
	// Handle cd command specially
	if strings.HasPrefix(cmd, "cd ") {
		path := strings.TrimSpace(cmd[3:])
		if path == "" {
			if homeDir, err := os.UserHomeDir(); err == nil {
				path = homeDir
			}
		} else if strings.HasPrefix(path, "~") {
			if homeDir, err := os.UserHomeDir(); err == nil {
				path = strings.Replace(path, "~", homeDir, 1)
			}
		} else if !filepath.IsAbs(path) {
			path = filepath.Join(c.config.CurrentDirectory, path)
		}

		if info, err := os.Stat(path); err == nil && info.IsDir() {
			if absPath, err := filepath.Abs(path); err == nil {
				c.config.CurrentDirectory = absPath
				return fmt.Sprintf("Changed directory to: %s", c.config.CurrentDirectory)
			}
		}
		return fmt.Sprintf("cd: %s: No such file or directory", path)
	}

	// Execute other commands
	var execCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		execCmd = exec.Command("cmd", "/C", cmd)
	} else {
		execCmd = exec.Command("sh", "-c", cmd)
	}

	execCmd.Dir = c.config.CurrentDirectory

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	output, err := execCmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "Command timed out (30s limit)"
		}
		return fmt.Sprintf("Error executing command: %v\nOutput: %s", err, string(output))
	}

	if len(output) == 0 {
		return fmt.Sprintf("Command executed (exit code: %d)", execCmd.ProcessState.ExitCode())
	}

	return string(output)
}

func (c *Client) executeADBCommand(cmd string) {
	if c.adbProcess == nil || c.adbProcess.ProcessState != nil {
		if !c.startADBShell() {
			c.sendOutput("Failed to start ADB shell\n" + c.getPrompt())
			return
		}
	}

	// Send command to ADB shell
	if c.adbStdin != nil {
		c.adbStdin.Write([]byte(cmd + "\n"))
	}
}

func (c *Client) startADBShell() bool {
	cmd := exec.Command(c.config.ADBPath, "shell")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("Failed to create stdin pipe: %v", err)
		return false
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to create stdout pipe: %v", err)
		return false
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start ADB shell: %v", err)
		return false
	}

	c.adbProcess = cmd
	c.adbStdin = stdin
	c.adbStdout = stdout

	// Start reading ADB output
	go c.readADBOutput()

	fmt.Println("ADB shell process started")
	return true
}

func (c *Client) readADBOutput() {
	reader := bufio.NewReader(c.adbStdout)

	for c.running && c.adbProcess != nil && c.adbProcess.ProcessState == nil {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF && c.running {
				log.Printf("Error reading ADB output: %v", err)
			}
			break
		}

		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")

		if line != "" {
			c.sendOutput(line)
		}
	}
}

func (c *Client) sendOutput(output string) {
	data := map[string]interface{}{
		"output": output,
	}
	eventData := []interface{}{"output_from_client", data}

	message, err := json.Marshal(eventData)
	if err != nil {
		log.Printf("Failed to marshal output: %v", err)
		return
	}

	fullMessage := "42" + string(message) // SocketIO event message type
	c.sendMessage(fullMessage)
}

func (c *Client) sendMessage(message string) {
	if c.conn != nil {
		c.conn.WriteMessage(websocket.TextMessage, []byte(message))
	}
}

func (c *Client) getPrompt() string {
	if c.config.CurrentMode == "local" {
		return fmt.Sprintf("local:%s$ ", filepath.Base(c.config.CurrentDirectory))
	}
	return "adb$ "
}

func findADB() string {
	// Get current executable directory
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	execDir := filepath.Dir(execPath)

	// Check local adb folder first
	localPaths := []string{
		filepath.Join(execDir, "adb", "adb"),
		filepath.Join(execDir, "adb", "adb.exe"),
		filepath.Join(execDir, "platform-tools", "adb"),
		filepath.Join(execDir, "platform-tools", "adb.exe"),
	}

	for _, path := range localPaths {
		if _, err := os.Stat(path); err == nil {
			// Test if ADB works
			if cmd := exec.Command(path, "version"); cmd.Run() == nil {
				fmt.Printf("Using local ADB: %s\n", path)
				return path
			}
		}
	}

	// Check system PATH
	if cmd := exec.Command("adb", "version"); cmd.Run() == nil {
		fmt.Println("Using system ADB from PATH")
		return "adb"
	}

	return ""
}