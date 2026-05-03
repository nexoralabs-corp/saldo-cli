package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func writeJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func writeHuman(format string, args ...any) error {
	_, err := fmt.Fprintf(os.Stdout, format, args...)
	return err
}

func readAll(r io.Reader) (string, error) {
	var builder strings.Builder
	_, err := io.Copy(&builder, r)
	if err != nil {
		return "", err
	}
	return builder.String(), nil
}

