package main

import (
	"context"
	"fmt"
	"time"

	"github.com/shivavenkatesh/moneta/pkg/types"
	"github.com/spf13/cobra"
)

var (
	indexLanguage string
)

var indexCmd = &cobra.Command{
	Use:   "index <path>",
	Short: "Index a file or directory",
	Long: `Index a file or directory to make its contents searchable. Files are
chunked intelligently (respecting function boundaries for code) and
stored with embeddings for semantic search.

Supported file types:
  Code: .go, .py, .js, .ts, .rs, .java, .c, .cpp, .rb, .php, .swift
  Text: .md, .txt, .yaml, .json, .toml, .sql, .sh

Ignored by default:
  .git, node_modules, vendor, __pycache__, .venv

Examples:
  moneta index ./src
  moneta index ./README.md
  moneta index . --project myapp`,
	Args: cobra.ExactArgs(1),
	RunE: runIndex,
}

func init() {
	indexCmd.Flags().StringVarP(&indexLanguage, "lang", "l", "", "Override language detection")
}

func runIndex(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	path := args[0]

	svc, err := initService()
	if err != nil {
		return err
	}
	defer svc.Close()

	fmt.Printf("Indexing %s...\n", path)
	start := time.Now()

	req := types.IndexRequest{
		Path:     path,
		Project:  getProject(),
		Language: indexLanguage,
	}

	count, err := svc.Index(ctx, req)
	if err != nil {
		return fmt.Errorf("indexing failed: %w", err)
	}

	elapsed := time.Since(start)
	fmt.Printf("Indexed %d chunks in %s\n", count, elapsed.Round(time.Millisecond))

	return nil
}
