package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandVersion(t *testing.T) {
	cmd := NewRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute --version: %v", err)
	}
	if got := strings.TrimSpace(output.String()); got != "saldo version "+Version {
		t.Fatalf("unexpected version output: %q", got)
	}
}
