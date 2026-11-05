package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/archictl/internal/config"
)

// Injected at link time via -ldflags.
var (
	version   = "dev"
	buildDate = "unknown"
	gitCommit = "unknown"
)

// Exit codes (PRODUCT_REQUIREMENTS.md section 9.3).
const (
	exitOK      = 0
	exitRefused = 1 // understood but refused: validation, ambiguity, unknown reference
	exitFatal   = 2 // usage, configuration or I/O error
)

// refusedError marks a domain-level refusal so Execute maps it to exit code 1.
type refusedError struct{ msg string }

func (e *refusedError) Error() string { return e.msg }

func refused(format string, a ...any) error { return &refusedError{fmt.Sprintf(format, a...)} }

// cfg is populated from global flags and passed into subcommands.
var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "archictl",
	Short: "Deterministic, token-cheap editing of Archi (.archimate) models for AI agents",
	Long: `archictl converts Archi models to a compact YAML view and back, and offers
validated, deterministic edit commands. It is a local batch tool: no network,
no sockets, no daemon.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute is the entry point called from main.
func Execute() {
	err := rootCmd.Execute()
	if err == nil {
		os.Exit(exitOK)
	}
	fmt.Fprintln(os.Stderr, "[ERROR]", err)
	var r *refusedError
	if errors.As(err, &r) {
		os.Exit(exitRefused)
	}
	os.Exit(exitFatal)
}

func init() {
	f := rootCmd.PersistentFlags()
	f.BoolVarP(&cfg.Verbose, "verbose", "v", false, "Enable debug-level logging to stderr")
	f.BoolVar(&cfg.NoColor, "no-color", false, "Disable ANSI colour in terminal output")
	f.BoolVar(&cfg.JSON, "json", false, "Write machine-readable JSON to stdout where a command supports it")
}
