// Moneta - A local-first code memory system
// Inspired by Supermemory, optimized for local development
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"

	// Global flags
	dataDir string
	project string
	verbose bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "moneta",
	Short: "Local-first code memory system",
	Long: `Moneta is a local-first code memory system that helps you store and retrieve
code context, patterns, and decisions using semantic search.

It runs entirely on your machine using local embeddings (Ollama) and SQLite
for storage, making it fast, private, and always available.

Examples:
  # Add a memory
  moneta add "We use Repository pattern for database access" --type pattern

  # Search for relevant memories
  moneta search "how do we access the database"

  # Index a codebase
  moneta index ./src --project myapp

  # Start the server for Claude Code integration
  moneta serve`,
	Version: Version,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "", "Data directory (default: ~/.moneta)")
	rootCmd.PersistentFlags().StringVarP(&project, "project", "p", "", "Project name (default: current directory name)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Add subcommands
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(statsCmd)
}
