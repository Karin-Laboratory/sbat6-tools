package policy

import "fmt"

var operations = map[string]string{
	"ping": "AT",
	"info": "AT+GMR",
}

func CommandFor(operation string) (string, error) {
	cmd, ok := operations[operation]
	if !ok {
		return "", fmt.Errorf("operation %q is not allowlisted", operation)
	}
	return cmd, nil
}
