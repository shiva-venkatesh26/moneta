package main

import (
	"context"
	"fmt"

	"github.com/shivavenkatesh/moneta/internal/store"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List memories",
	Long: `List all memories in the current project.

Examples:
  moneta list
  moneta list --type pattern
  moneta list --limit 20`,
	RunE: runList,
}

var (
	listLimit int
	listType  string
)

func init() {
	listCmd.Flags().IntVarP(&listLimit, "limit", "n", 20, "Maximum results")
	listCmd.Flags().StringVar(&listType, "type", "", "Filter by type")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	svc, err := initService()
	if err != nil {
		return err
	}
	defer svc.Close()

	opts := store.ListOptions{
		Project:    getProject(),
		Limit:      listLimit,
		Descending: true,
		OrderBy:    "created_at",
	}

	memories, err := svc.List(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to list memories: %w", err)
	}

	if len(memories) == 0 {
		fmt.Println("No memories found")
		return nil
	}

	fmt.Printf("Memories in project '%s':\n\n", getProject())
	for _, m := range memories {
		fmt.Printf("  [%s] %s\n", formatType(m.Type), truncate(m.Content, 80))
		fmt.Printf("    ID: %s\n\n", m.ID)
	}

	return nil
}

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a memory",
	Long: `Delete a memory by its ID.

Examples:
  moneta delete abc123
  moneta delete --all  # Delete all memories in project`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDelete,
}

var deleteAll bool

func init() {
	deleteCmd.Flags().BoolVar(&deleteAll, "all", false, "Delete all memories in project")
}

func runDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	svc, err := initService()
	if err != nil {
		return err
	}
	defer svc.Close()

	if deleteAll {
		if err := svc.DeleteByProject(ctx, getProject()); err != nil {
			return fmt.Errorf("failed to delete memories: %w", err)
		}
		fmt.Printf("Deleted all memories in project '%s'\n", getProject())
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("memory ID required (or use --all)")
	}

	id := args[0]
	if err := svc.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	fmt.Printf("Deleted: %s\n", id)
	return nil
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show statistics",
	Long: `Show statistics about stored memories.

Examples:
  moneta stats`,
	RunE: runStats,
}

func runStats(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	svc, err := initService()
	if err != nil {
		return err
	}
	defer svc.Close()

	stats, err := svc.Stats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Println("Moneta Statistics")
	fmt.Println("─────────────────")
	fmt.Printf("Total memories:  %d\n", stats.TotalMemories)
	fmt.Printf("Projects:        %d\n", stats.ProjectCount)
	fmt.Printf("Embedding model: %s\n", stats.EmbeddingModel)
	fmt.Printf("Storage size:    %.2f MB\n", float64(stats.StorageBytes)/1024/1024)
	fmt.Println()

	if len(stats.MemoriesByType) > 0 {
		fmt.Println("By type:")
		for t, count := range stats.MemoriesByType {
			fmt.Printf("  %-15s %d\n", t, count)
		}
	}

	return nil
}
