package vendor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

var ErrBusy = errors.New("ESP32 vendor path is busy")

type Runner struct {
	Frontend string
	LockDir  string
	Timeout  time.Duration
	MaxBytes int
}

type Result struct {
	Output   string
	ExitCode int
}

func (r Runner) Run(command string) (Result, error) {
	if r.Frontend == "" || r.LockDir == "" || r.Timeout <= 0 || r.MaxBytes <= 0 {
		return Result{}, errors.New("invalid runner configuration")
	}

	if err := os.Mkdir(r.LockDir, 0700); err != nil {
		if os.IsExist(err) {
			return Result{}, ErrBusy
		}
		return Result{}, fmt.Errorf("create lock: %w", err)
	}
	defer os.Remove(r.LockDir)

	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.Frontend, "--atcmd", command)
	out, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return Result{}, fmt.Errorf("vendor frontend timed out after %s", r.Timeout)
	}
	if len(out) > r.MaxBytes {
		return Result{}, fmt.Errorf("vendor output exceeded safe broker limit: %d > %d bytes; long responses remain disabled", len(out), r.MaxBytes)
	}

	result := Result{Output: strings.TrimSpace(string(out))}
	if err == nil {
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, fmt.Errorf("vendor frontend exited with status %d", result.ExitCode)
	}
	return result, fmt.Errorf("run vendor frontend: %w", err)
}
