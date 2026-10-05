package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Karin-Laboratory/sbat6-tools/esp32/broker/internal/api"
)

func main() {
	socket := flag.String("socket", "/var/run/esp32d.sock", "esp32d Unix socket")
	timeout := flag.Duration("timeout", 3*time.Second, "broker request timeout")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: esp32ctl-broker [flags] ping|info")
		os.Exit(64)
	}
	op := flag.Arg(0)
	if op != "ping" && op != "info" {
		fmt.Fprintf(os.Stderr, "esp32ctl-broker: operation %q is not allowed\n", op)
		os.Exit(77)
	}

	conn, err := net.DialTimeout("unix", *socket, *timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "esp32ctl-broker: connect: %v\n", err)
		os.Exit(69)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(*timeout))

	if err := json.NewEncoder(conn).Encode(api.Request{Operation: op}); err != nil {
		fmt.Fprintf(os.Stderr, "esp32ctl-broker: send: %v\n", err)
		os.Exit(74)
	}

	var resp api.Response
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&resp); err != nil {
		fmt.Fprintf(os.Stderr, "esp32ctl-broker: receive: %v\n", err)
		os.Exit(74)
	}
	if resp.Output != "" {
		fmt.Println(resp.Output)
	}
	if !resp.OK {
		fmt.Fprintf(os.Stderr, "esp32ctl-broker: %s\n", resp.Error)
		os.Exit(70)
	}
}
