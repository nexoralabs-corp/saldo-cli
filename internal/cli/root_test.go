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

func TestRootExposesFinancialManagementCommands(t *testing.T) {
	root := NewRootCommand()
	for _, name := range []string{"accounts", "categories", "tags", "credit-cards", "loans", "subscriptions", "budgets", "import"} {
		command, _, err := root.Find([]string{name})
		if err != nil || command == root {
			t.Fatalf("missing %s command: %v", name, err)
		}
	}
}
