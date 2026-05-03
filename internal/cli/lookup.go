package cli

import (
	"context"

	"github.com/spf13/cobra"
)

func newCategoriesCommand(state *appState) *cobra.Command {
	var query string
	var categoryType string
	cmd := &cobra.Command{
		Use:   "categories",
		Short: "Search categories",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List matching categories",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, _, err := requireSessionClient(state)
			if err != nil {
				return err
			}
			var data struct {
				SearchCategories []category `json:"searchCategories"`
			}
			vars := map[string]any{"query": query}
			if categoryType != "" {
				vars["type"] = categoryType
			}
			if err := client.Do(context.Background(), categoriesQuery, vars, &data); err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(data.SearchCategories)
			}
			for _, category := range data.SearchCategories {
				if err := writeHuman("%s\t%s\t%s\n", category.ID, category.Name, category.Type); err != nil {
					return err
				}
			}
			return nil
		},
	})
	cmd.PersistentFlags().StringVar(&query, "query", "", "category search text")
	cmd.PersistentFlags().StringVar(&categoryType, "type", "", "category type: INCOME, EXPENSE, or BOTH")
	return cmd
}

func newTagsCommand(state *appState) *cobra.Command {
	var query string
	cmd := &cobra.Command{Use: "tags", Short: "Search tags"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List matching tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, _, err := requireSessionClient(state)
			if err != nil {
				return err
			}
			var data struct {
				MyTags []tag `json:"myTags"`
			}
			if err := client.Do(context.Background(), tagsQuery, map[string]any{"query": query}, &data); err != nil {
				return err
			}
			if state.jsonOutput {
				return writeJSON(data.MyTags)
			}
			for _, tag := range data.MyTags {
				if err := writeHuman("%s\t%s\n", tag.ID, tag.Name); err != nil {
					return err
				}
			}
			return nil
		},
	})
	cmd.PersistentFlags().StringVar(&query, "query", "", "tag search text")
	return cmd
}

const categoryFields = `id name type color`
const tagFields = `id name color`

const categoriesQuery = `
query Categories($query: String!, $type: String) {
  searchCategories(query: $query, type: $type) { ` + categoryFields + ` }
}`

const tagsQuery = `
query Tags($query: String!) {
  myTags(query: $query) { ` + tagFields + ` }
}`

