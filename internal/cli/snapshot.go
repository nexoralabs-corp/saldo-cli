package cli

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
)

func newSnapshotCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{Use: "snapshot", Short: "Generate agent context snapshots"}
	cmd.AddCommand(newSnapshotAICommand(state))
	return cmd
}

func newSnapshotAICommand(state *appState) *cobra.Command {
	var from string
	var to string
	var sections []string
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "Generate an AI-friendly financial snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			if from == "" || to == "" {
				return cmd.Help()
			}
			client, _, _, err := requireSessionClient(state)
			if err != nil {
				return err
			}
			input := map[string]any{"startDate": from, "endDate": to}
			if len(sections) > 0 {
				input["sections"] = snapshotSections(sections)
			}
			var data struct {
				AiSnapshot struct {
					Filename string `json:"filename"`
					Markdown string `json:"markdown"`
				} `json:"aiSnapshot"`
			}
			if err := client.Do(context.Background(), aiSnapshotQuery, map[string]any{"input": input}, &data); err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(data.AiSnapshot)
			}
			return writeHuman("%s\n", data.AiSnapshot.Markdown)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "start date in YYYY-MM-DD format")
	cmd.Flags().StringVar(&to, "to", "", "end date in YYYY-MM-DD format")
	cmd.Flags().StringArrayVar(&sections, "section", nil, "snapshot section; may be repeated")
	return cmd
}

func snapshotSections(sections []string) map[string]any {
	enabled := map[string]any{
		"netWorth":      false,
		"transactions":  false,
		"subscriptions": false,
		"loans":         false,
		"creditCards":   false,
		"budgets":       false,
	}
	for _, section := range sections {
		switch strings.ToLower(strings.ReplaceAll(section, "-", "_")) {
		case "net_worth", "networth":
			enabled["netWorth"] = true
		case "transactions":
			enabled["transactions"] = true
		case "subscriptions":
			enabled["subscriptions"] = true
		case "loans":
			enabled["loans"] = true
		case "credit_cards", "creditcards":
			enabled["creditCards"] = true
		case "budgets":
			enabled["budgets"] = true
		}
	}
	return enabled
}

const aiSnapshotQuery = `
query AiSnapshot($input: AiSnapshotInput!) {
  aiSnapshot(input: $input) {
    filename
    markdown
  }
}`
