package cli

import (
	"context"
	"fmt"
	"strings"

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
	cmd.AddCommand(newCategoryCreateCommand(state), newCategoryUpdateCommand(state), newCategoryDeleteCommand(state))
	return cmd
}

func newCategoryCreateCommand(state *appState) *cobra.Command {
	var name, kind, color, parentID string
	cmd := &cobra.Command{Use: "create", Short: "Create a category", RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("--name is required")
		}
		kind = strings.ToUpper(kind)
		if kind != "INCOME" && kind != "EXPENSE" && kind != "BOTH" {
			return fmt.Errorf("--type must be INCOME, EXPENSE, or BOTH")
		}
		input := map[string]any{"name": name, "type": kind, "color": color}
		if parentID != "" {
			input["parentId"] = parentID
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreateCategory category `json:"createCategory"`
		}
		if err = client.Do(context.Background(), createCategoryMutation, map[string]any{"input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreateCategory)
		}
		return writeHuman("Created category %s\n", data.CreateCategory.ID)
	}}
	cmd.Flags().StringVar(&name, "name", "", "category name")
	cmd.Flags().StringVar(&kind, "type", "BOTH", "INCOME, EXPENSE, or BOTH")
	cmd.Flags().StringVar(&color, "color", "#17171c", "hex color")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "parent category ID")
	return cmd
}

func newCategoryUpdateCommand(state *appState) *cobra.Command {
	var name, kind, color, parentID string
	cmd := &cobra.Command{Use: "update <id>", Short: "Update a category", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		input := map[string]any{}
		for _, v := range []struct {
			flag, key string
			value     any
		}{{"name", "name", name}, {"type", "type", strings.ToUpper(kind)}, {"color", "color", color}, {"parent-id", "parentId", parentID}} {
			if cmd.Flags().Changed(v.flag) {
				input[v.key] = v.value
			}
		}
		if len(input) == 0 {
			return fmt.Errorf("at least one field must be provided")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			UpdateCategory category `json:"updateCategory"`
		}
		if err = client.Do(context.Background(), updateCategoryMutation, map[string]any{"id": args[0], "input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.UpdateCategory)
		}
		return writeHuman("Updated category %s\n", data.UpdateCategory.ID)
	}}
	cmd.Flags().StringVar(&name, "name", "", "category name")
	cmd.Flags().StringVar(&kind, "type", "", "INCOME, EXPENSE, or BOTH")
	cmd.Flags().StringVar(&color, "color", "", "hex color")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "parent category ID")
	return cmd
}

func newCategoryDeleteCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "delete <id>", Short: "Delete a category", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return deleteLookup(state, deleteCategoryMutation, "deleteCategory", args[0])
	}}
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
	cmd.AddCommand(newTagCreateCommand(state), newTagUpdateCommand(state), newTagDeleteCommand(state))
	return cmd
}

func newTagCreateCommand(state *appState) *cobra.Command {
	var name, color string
	cmd := &cobra.Command{Use: "create", Short: "Create a tag", RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("--name is required")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			CreateTag tag `json:"createTag"`
		}
		if err = client.Do(context.Background(), createTagMutation, map[string]any{"input": map[string]any{"name": name, "color": color}}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.CreateTag)
		}
		return writeHuman("Created tag %s\n", data.CreateTag.ID)
	}}
	cmd.Flags().StringVar(&name, "name", "", "tag name")
	cmd.Flags().StringVar(&color, "color", "#17171c", "hex color")
	return cmd
}
func newTagUpdateCommand(state *appState) *cobra.Command {
	var name, color string
	cmd := &cobra.Command{Use: "update <id>", Short: "Update a tag", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		input := map[string]any{}
		if cmd.Flags().Changed("name") {
			input["name"] = name
		}
		if cmd.Flags().Changed("color") {
			input["color"] = color
		}
		if len(input) == 0 {
			return fmt.Errorf("at least one field must be provided")
		}
		client, _, _, err := requireSessionClient(state)
		if err != nil {
			return err
		}
		var data struct {
			UpdateTag tag `json:"updateTag"`
		}
		if err = client.Do(context.Background(), updateTagMutation, map[string]any{"id": args[0], "input": input}, &data); err != nil {
			return err
		}
		if state.jsonOutput {
			return writeJSON(data.UpdateTag)
		}
		return writeHuman("Updated tag %s\n", data.UpdateTag.ID)
	}}
	cmd.Flags().StringVar(&name, "name", "", "tag name")
	cmd.Flags().StringVar(&color, "color", "", "hex color")
	return cmd
}
func newTagDeleteCommand(state *appState) *cobra.Command {
	return &cobra.Command{Use: "delete <id>", Short: "Delete a tag", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return deleteLookup(state, deleteTagMutation, "deleteTag", args[0])
	}}
}

func deleteLookup(state *appState, mutation, field, id string) error {
	client, _, _, err := requireSessionClient(state)
	if err != nil {
		return err
	}
	data := map[string]bool{}
	if err = client.Do(context.Background(), mutation, map[string]any{"id": id}, &data); err != nil {
		return err
	}
	if state.jsonOutput {
		return writeJSON(map[string]any{"deleted": data[field], "id": id})
	}
	return writeHuman("Deleted %s\n", id)
}

const categoryFields = `id name type color parentId`
const tagFields = `id name color`

const categoriesQuery = `
query Categories($query: String!, $type: String) {
  searchCategories(query: $query, type: $type) { ` + categoryFields + ` }
}`

const tagsQuery = `
query Tags($query: String!) {
  myTags(query: $query) { ` + tagFields + ` }
}`

const createCategoryMutation = `mutation($input: CreateCategoryInput!){createCategory(input:$input){` + categoryFields + `}}`
const updateCategoryMutation = `mutation($id:ID!,$input:UpdateCategoryInput!){updateCategory(id:$id,input:$input){` + categoryFields + `}}`
const deleteCategoryMutation = `mutation($id:ID!){deleteCategory(id:$id)}`
const createTagMutation = `mutation($input:CreateTagInput!){createTag(input:$input){` + tagFields + `}}`
const updateTagMutation = `mutation($id:ID!,$input:UpdateTagInput!){updateTag(id:$id,input:$input){` + tagFields + `}}`
const deleteTagMutation = `mutation($id:ID!){deleteTag(id:$id)}`
