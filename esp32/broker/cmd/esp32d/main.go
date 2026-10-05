package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Karin-Laboratory/sbat6-tools/esp32/broker/internal/api"
	"github.com/Karin-Laboratory/sbat6-tools/esp32/broker/internal/policy"
	"github.com/Karin-Laboratory/sbat6-tools/esp32/broker/internal/vendor"
)

const maxRequestBytes = 1024

func main() {
	socket := flag.String("socket", "/var/run/esp32d.sock", "Unix socket path")
	frontend := flag.String("frontend", "/usr/sbin/esp32_controller", "vendor ESP32 frontend")
	lockDir := flag.String("lock", "/tmp/esp32ctl.lock", "shared inter-process lock directory")
	timeout := flag.Duration("timeout", 5*time.Second, "vendor command timeout")
	maxOutput := flag.Int("max-output", 240, "maximum accepted vendor output bytes")
	flag.Parse()

	if err := prepareSocket(*socket); err != nil {
		log.Fatal(err)
	}
	ln, err := net.Listen("unix", *socket)
	if err != nil {
		log.Fatalf("listen %s: %v", *socket, err)
	}
	defer func() {
		_ = ln.Close()
		_ = os.Remove(*socket)
	}()
	if err := os.Chmod(*socket, 0660); err != nil {
		log.Fatalf("chmod socket: %v", err)
	}

	runner := vendor.Runner{Frontend: *frontend, LockDir: *lockDir, Timeout: *timeout, MaxBytes: *maxOutput}
	log.Printf("esp32d listening on %s", *socket)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		// Intentionally serial: this process is the ownership boundary.
		handle(conn, runner)
	}
}

func prepareSocket(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}
	st, err := os.Lstat(path)
	if err == nil {
		if st.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("refusing to remove non-socket path %s", path)
		}
		return os.Remove(path)
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("inspect socket path: %w", err)
	}
	return nil
}

func handle(conn net.Conn, runner vendor.Runner) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	reader := bufio.NewReaderSize(conn, maxRequestBytes+1)
	line, err := reader.ReadBytes('\n')
	if err != nil || len(line) > maxRequestBytes {
		writeResponse(conn, api.Response{OK: false, Error: "invalid, incomplete, or oversized request"})
		return
	}

	var req api.Request
	if err := json.Unmarshal(line, &req); err != nil {
		writeResponse(conn, api.Response{OK: false, Error: "invalid JSON"})
		return
	}

	command, err := policy.CommandFor(req.Operation)
	if err != nil {
		writeResponse(conn, api.Response{ID: req.ID, OK: false, Error: err.Error()})
		return
	}

	result, err := runner.Run(command)
	if err != nil {
		writeResponse(conn, api.Response{ID: req.ID, OK: false, Output: result.Output, Error: err.Error(), ExitCode: result.ExitCode})
		return
	}
	writeResponse(conn, api.Response{ID: req.ID, OK: true, Output: result.Output})
}

func writeResponse(conn net.Conn, resp api.Response) {
	_ = json.NewEncoder(conn).Encode(resp)
}
