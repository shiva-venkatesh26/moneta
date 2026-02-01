package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/shivavenkatesh/moneta/pkg/types"
	"github.com/spf13/cobra"
)

var (
	searchLimit     int
	searchThreshold float32
	searchType      string
	searchJSON      bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for memories",
	Long: `Search for relevant memories using semantic search. The query is converted
to an embedding and compared against stored memories using cosine similarity.

Examples:
  moneta search "how do we handle authentication"
  moneta search "database patterns" --limit 5
  moneta search "error handling" --type gotcha
  moneta search "API design" --threshold 0.7`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 10, "Maximum results to return")
	searchCmd.Flags().Float32VarP(&searchThreshold, "threshold", "t", 0.5, "Minimum similarity threshold (0-1)")
	searchCmd.Flags().StringVar(&searchType, "type", "", "Filter by memory type")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output as JSON")
}

func runSearch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	query := strings.Join(args, " ")
	if query == "" {
		return fmt.Errorf("query is required")
	}

	svc, err := initService()
	if err != nil {
		return err
	}
	defer svc.Close()

	req := types.SearchRequest{
		Query:     query,
		Project:   getProject(),
		Limit:     searchLimit,
		Threshold: searchThreshold,
	}

	if searchType != "" {
		req.Type = types.MemoryType(searchType)
	}

	resp, err := svc.Search(ctx, req)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(resp.Results) == 0 {
		fmt.Println("No results found")
		return nil
	}

	if searchJSON {
		return printJSON(resp)
	}

	// Print results
	fmt.Printf("Found %d results (%.0fms):\n\n", resp.Total, float64(resp.Timing))

	for i, result := range resp.Results {
		fmt.Printf("%d. [%.2f] %s\n", i+1, result.Similarity, formatType(result.Memory.Type))
		fmt.Printf("   %s\n", formatContent(result.Memory.Content))
		if result.Memory.FilePath != "" {
			fmt.Printf("   File: %s\n", result.Memory.FilePath)
		}
		fmt.Println()
	}

	return nil
}

func formatType(t types.MemoryType) string {
	colors := map[types.MemoryType]string{
		types.TypeArchitecture: "\033[34m", // Blue
		types.TypePattern:      "\033[32m", // Green
		types.TypeDecision:     "\033[33m", // Yellow
		types.TypeGotcha:       "\033[31m", // Red
		types.TypeContext:      "\033[37m", // White
		types.TypePreference:   "\033[35m", // Magenta
	}
	reset := "\033[0m"
	color := colors[t]
	if color == "" {
		color = "\033[37m"
	}
	return fmt.Sprintf("%s%s%s", color, t, reset)
}

func formatContent(content string) string {
	// Truncate and clean up
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.Join(strings.Fields(content), " ") // Normalize whitespace
	if len(content) > 200 {
		content = content[:197] + "..."
	}
	return content
}

func printJSON(v interface{}) error {
	// Simple JSON output (could use encoding/json for pretty print)
	fmt.Printf("%+v\n", v)
	return nil
}
