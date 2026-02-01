package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/shivavenkatesh/moneta/pkg/types"
	"github.com/spf13/cobra"
)

var (
	addType     string
	addFilePath string
	addLanguage string
	addMetadata []string
)

var addCmd = &cobra.Command{
	Use:   "add <content>",
	Short: "Add a memory",
	Long: `Add a new memory to the store. Memories can be code patterns, architecture
decisions, gotchas, or any context you want to remember.

Types:
  architecture  - High-level design decisions
  pattern       - Code patterns and conventions
  decision      - Why something was done
  gotcha        - Bugs, edge cases, warnings
  context       - General context (default)
  preference    - User coding preferences

Examples:
  moneta add "We use Repository pattern for DB access" --type pattern
  moneta add "Always validate user input before SQL queries" --type gotcha
  moneta add "Chose PostgreSQL for ACID transactions" --type decision`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringVarP(&addType, "type", "t", "context", "Memory type (architecture, pattern, decision, gotcha, context, preference)")
	addCmd.Flags().StringVarP(&addFilePath, "file", "f", "", "Associated file path")
	addCmd.Flags().StringVarP(&addLanguage, "lang", "l", "", "Programming language")
	addCmd.Flags().StringArrayVarP(&addMetadata, "meta", "m", nil, "Metadata as key=value pairs")
}

func runAdd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	content := strings.Join(args, " ")
	if content == "" {
		return fmt.Errorf("content is required")
	}

	svc, err := initService()
	if err != nil {
		return err
	}
	defer svc.Close()

	// Parse metadata
	metadata := make(map[string]string)
	for _, m := range addMetadata {
		parts := strings.SplitN(m, "=", 2)
		if len(parts) == 2 {
			metadata[parts[0]] = parts[1]
		}
	}

	req := types.AddMemoryRequest{
		Content:  content,
		Project:  getProject(),
		Type:     types.MemoryType(addType),
		FilePath: addFilePath,
		Language: addLanguage,
		Metadata: metadata,
	}

	memory, err := svc.Add(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to add memory: %w", err)
	}

	if verbose {
		fmt.Printf("Added memory:\n")
		fmt.Printf("  ID:      %s\n", memory.ID)
		fmt.Printf("  Type:    %s\n", memory.Type)
		fmt.Printf("  Project: %s\n", memory.Project)
		fmt.Printf("  Content: %s\n", truncate(memory.Content, 100))
	} else {
		fmt.Printf("Added: %s\n", memory.ID)
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func getProject() string {
	if project != "" {
		return project
	}
	// Default to current directory name
	dir, err := os.Getwd()
	if err != nil {
		return "default"
	}
	parts := strings.Split(dir, string(os.PathSeparator))
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "default"
}
