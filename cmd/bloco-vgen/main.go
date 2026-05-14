package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"bloco-vgen/internal/cli"
	"bloco-vgen/internal/config"
	"bloco-vgen/pkg/errors"

	"github.com/charmbracelet/fang"
)

// Version information (set during build)
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func main() {
	// Setup graceful shutdown
	ctx, cancel := setupGracefulShutdown()
	defer cancel()

	// Load configuration
	cfg := config.DefaultConfig()
	cfg.LoadFromEnvironment()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Create CLI application
	app := cli.NewApplication(cfg, Version, GitCommit, BuildTime)

	// Execute with fang for smooth animations and signal handling
	if err := fang.Execute(
		ctx,
		app.GetRootCommand(),
		fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),
	); err != nil {
		handleError(err)
		os.Exit(1)
	}
}

// setupGracefulShutdown sets up graceful shutdown handling
func setupGracefulShutdown() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Fprintf(os.Stderr, "\nReceived interrupt signal, shutting down gracefully...\n")
		cancel()
	}()

	return ctx, cancel
}

// handleError handles application errors with appropriate formatting
func handleError(err error) {
	if blocoErr, ok := err.(*errors.BlocoError); ok {
		// Handle structured errors
		fmt.Fprintf(os.Stderr, "Error: %s\n", blocoErr.Error())

		// Show context in verbose mode
		if len(blocoErr.Context) > 0 {
			fmt.Fprintf(os.Stderr, "Context:\n")
			for key, value := range blocoErr.Context {
				fmt.Fprintf(os.Stderr, "  %s: %v\n", key, value)
			}
		}

		// Show stack trace in debug mode
		if os.Getenv("BLOCO_DEBUG") != "" && len(blocoErr.Stack) > 0 {
			fmt.Fprintf(os.Stderr, "Stack trace:\n")
			for _, frame := range blocoErr.Stack {
				fmt.Fprintf(os.Stderr, "  %s\n", frame)
			}
		}
	} else {
		// Handle generic errors
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}
