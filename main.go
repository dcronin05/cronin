package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Payload struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

func main() {
	bgFlag := flag.Bool("bg", false, "Run in background")
	ttyFlag := flag.String("tty", "", "TTY device")
	fileFlag := flag.String("f", "", "File payload")
	clipFlag := flag.Bool("clip", false, "Include clipboard contents")
	liveFlag := flag.Bool("live", false, "Stream internal thoughts and tool executions in real-time")
	flag.Parse()

	if *bgFlag {
		runBackground(*ttyFlag, flag.Args(), *liveFlag)
		return
	}

	msg := strings.Join(flag.Args(), " ")

	if *fileFlag != "" {
		fileBytes, err := os.ReadFile(*fileFlag)
		if err == nil {
			if msg != "" { msg += "\n\n" }
			msg += "--- File: " + *fileFlag + " ---\n" + string(fileBytes)
		}
	}

	if *clipFlag {
		clipBytes := getClipboard()
		if len(clipBytes) > 0 {
			if msg != "" { msg += "\n\n" }
			msg += "--- Clipboard ---\n" + string(clipBytes)
		}
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		stdinBytes, err := io.ReadAll(os.Stdin)
		if err == nil && len(stdinBytes) > 0 {
			if msg != "" { msg += "\n\n" }
			msg += "--- Piped Data ---\n" + string(stdinBytes)
		}
	}

	msg = strings.TrimSpace(strings.ReplaceAll(msg, "\x00", ""))
	if msg == "" {
		fmt.Println("Usage: cronin [-f file.txt] [--clip] [--live] <message>")
		os.Exit(1)
	}

	ttyPath := ""
	b, _ := exec.Command("tty").Output()
	ttyPath = strings.TrimSpace(string(b))
	if ttyPath == "not a tty" || ttyPath == "" {
		// fallback using sh
		b, _ = exec.Command("sh", "-c", "tty 0>&2").Output()
		ttyPath = strings.TrimSpace(string(b))
	}
	if ttyPath == "not a tty" || ttyPath == "" {
		ttyPath = "/dev/tty"
	}

	exe, _ := os.Executable()
	
	bgArgs := []string{"--bg", "--tty", ttyPath}
	if *liveFlag {
		bgArgs = append(bgArgs, "--live")
	}
	bgArgs = append(bgArgs, msg)
	
	cmd := exec.Command(exe, bgArgs...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	err := cmd.Start()
	if err != nil {
		fmt.Println("Failed to start background process:", err)
		os.Exit(1)
	}

	fmt.Println("Task handed to Cronin! Working in the background...")
	os.Exit(0)
}

func runBackground(ttyPath string, args []string, isLive bool) {
	debugFile, _ := os.OpenFile("/tmp/cronin_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if debugFile != nil {
		defer debugFile.Close()
		debugFile.WriteString("--- Starting Background Process ---\n")
		debugFile.WriteString("Resolved ttyPath: " + ttyPath + "\n")
	}

	msg := strings.Join(args, " ")
	hostname, _ := os.Hostname()
	if hostname == "" { hostname = "UnknownDevice" }

	payload := map[string]string{
		"source":  hostname,
		"message": msg,
	}
	jsonData, _ := json.Marshal(payload)

	endpoint := os.Getenv("CRONIN_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8081/webhook/stream"
	}

	req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	token := os.Getenv("CRONIN_TOKEN")
	if token != "" {
		req.Header.Set("Authorization", "Bearer " + token)
	}

	if isLive {
		req.Header.Set("X-Live-Mode", "true")
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		if debugFile != nil { debugFile.WriteString("HTTP Request failed: " + err.Error() + "\n") }
		return
	}
	defer resp.Body.Close()

	if debugFile != nil { debugFile.WriteString("HTTP Request succeeded. Headers received.\n") }

	var tty *os.File
	if ttyPath != "" {
		tty, err = os.OpenFile(ttyPath, os.O_WRONLY, 0)
		if err != nil && debugFile != nil {
			debugFile.WriteString("Failed to open " + ttyPath + ": " + err.Error() + "\n")
		}
	}
	if tty == nil {
		tty, err = os.OpenFile("/dev/tty", os.O_WRONLY, 0)
		if err != nil && debugFile != nil {
			debugFile.WriteString("Failed to open /dev/tty: " + err.Error() + "\n")
		}
	}
	if tty == nil {
		if debugFile != nil { debugFile.WriteString("ABORTING: No TTY available to write to.\n") }
		return
	}
	defer tty.Close()

	if debugFile != nil { debugFile.WriteString("Successfully opened TTY. Writing header.\n") }
	tty.WriteString("\r\n\033[36m[Cronin is typing...]\033[0m\r\n")
	
	buffer := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			chunkBytes := buffer[:n]
			chunkBytes = bytes.ReplaceAll(chunkBytes, []byte("\x00"), []byte(""))
			chunkBytes = bytes.ReplaceAll(chunkBytes, []byte("\r"), []byte(""))
			chunkBytes = bytes.ReplaceAll(chunkBytes, []byte("\n"), []byte("\r\n"))
			tty.Write(chunkBytes)
		}
		if err != nil {
			if debugFile != nil { debugFile.WriteString("Reader closed/error: " + err.Error() + "\n") }
			break
		}
	}
	tty.WriteString("\r\n\033[36m[Cronin Finished]\033[0m\r\n")
	if debugFile != nil { debugFile.WriteString("Finished streaming.\n") }
}

func getClipboard() []byte {
	// Try Wayland
	if out, err := exec.Command("wl-paste").Output(); err == nil {
		return out
	}
	// Try X11 xclip
	if out, err := exec.Command("xclip", "-selection", "clipboard", "-o").Output(); err == nil {
		return out
	}
	// Try X11 xsel
	if out, err := exec.Command("xsel", "--clipboard", "--output").Output(); err == nil {
		return out
	}
	return nil
}
