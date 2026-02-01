package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/shivavenkatesh/moneta/internal/server"
	"github.com/spf13/cobra"
)

var (
	servePort int
	serveHost string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long: `Start the HTTP server for Claude Code integration and other clients.

The server exposes a REST API for:
  - Adding memories
  - Searching memories
  - Managing projects

This allows Claude Code plugins to connect to Moneta for persistent
memory across coding sessions.

Examples:
  moneta serve
  moneta serve --port 3456
  moneta serve --host 0.0.0.0 --port 8080`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 3456, "Port to listen on")
	serveCmd.Flags().StringVar(&serveHost, "host", "127.0.0.1", "Host to bind to")
}

func runServe(cmd *cobra.Command, args []string) error {
	svc, err := initService()
	if err != nil {
		return err
	}

	srv := server.New(svc, server.Config{
		Host: serveHost,
		Port: servePort,
	})

	// Handle graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-done
		fmt.Println("\nShutting down...")
		srv.Shutdown()
		svc.Close()
	}()

	addr := fmt.Sprintf("%s:%d", serveHost, servePort)
	fmt.Printf("Moneta server listening on http://%s\n", addr)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  POST /memory      - Add a memory")
	fmt.Println("  POST /search      - Search memories")
	fmt.Println("  GET  /memory/:id  - Get a memory")
	fmt.Println("  DELETE /memory/:id - Delete a memory")
	fmt.Println("  GET  /stats       - Get statistics")
	fmt.Println("  GET  /health      - Health check")

	return srv.Start()
}
