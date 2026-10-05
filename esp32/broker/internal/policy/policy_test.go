package policy

import "testing"

func TestAllowlist(t *testing.T) {
	tests := map[string]string{"ping": "AT", "info": "AT+GMR"}
	for op, want := range tests {
		got, err := CommandFor(op)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", op, err)
		}
		if got != want {
			t.Fatalf("%s: got %q want %q", op, got, want)
		}
	}
}

func TestRejectsUnknownAndStateChangingOperations(t *testing.T) {
	for _, op := range []string{"raw", "scan", "wifi-scan", "ble-scan", "reset", "flash", "ota", "configure", "AT+RST"} {
		if cmd, err := CommandFor(op); err == nil {
			t.Fatalf("%q unexpectedly allowed as %q", op, cmd)
		}
	}
}
