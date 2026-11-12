package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/ibmdocs/internal/config"
	"github.com/ibm-webmethods-continuous-delivery/7u-cli-tools/code/ibmdocs/internal/ibmdocs"
)

// errPartialFailure is returned when one or more topics fail but others succeed.
// Execute() maps this to exit code 1 without printing usage or the error string.
var errPartialFailure = errors.New("one or more topics failed")

// Injected at link time via -ldflags.
var (
	version   = "dev"
	buildDate = "unknown"
	gitCommit = "unknown"
)

// cfg is populated from global flags and passed into subcommands.
var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "ibmdocs",
	Short: "Fetch and cache IBM product documentation as agent-ready Markdown",
	Long: `ibmdocs fetches IBM product documentation from the IBM Docs public API,
caches it locally, and exports it as agent-ready Markdown.

Data folder, when --data is not given: $IWCD_IBMDOCS_DATA_FOLDER, else
$XDG_CACHE_HOME/ibmdocs-data, else ~/.cache/ibmdocs-data, else ./ibmdocs-data.
Run 'ibmdocs data-dir' to print the resolved folder.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if errors.Is(err, errPartialFailure) {
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "[ERROR]", err)
		os.Exit(1)
	}
}

func init() {
	f := rootCmd.PersistentFlags()

	f.StringVarP(&cfg.DataDir, "data", "d", "",
		"Path to the data folder (cache/, kb/, index.json). Default: env "+config.EnvDataFolder+
			", else $XDG_CACHE_HOME/ibmdocs-data, else ~/.cache/ibmdocs-data, else ./ibmdocs-data")

	f.StringVar(&cfg.Lang, "lang", config.DefaultLang,
		"Language code for IBM Docs API requests (e.g. en, fr, de, ja)")

	f.BoolVarP(&cfg.Verbose, "verbose", "v", false,
		"Enable debug-level logging to stderr")

	f.BoolVar(&cfg.NoColor, "no-color", false,
		"Disable ANSI colour in terminal output")

	f.BoolVar(&cfg.HTTPDebug, "http-debug", false,
		"Dump every HTTP request/response to <data>/debug/")

	f.StringVar(&cfg.CDNBaseURL, "cdn-base-url",
		envOr(config.EnvCDNBaseURL, config.DefaultCDNBaseURL),
		"IBM Docs CDN base URL (env: IBMDOCS_CDN_BASE_URL)")

	f.DurationVar(&cfg.Delay, "delay", config.DefaultDelay,
		"Inter-request delay for polite crawling (e.g. 200ms, 0 to disable)")

	f.DurationVar(&cfg.CacheTTL, "cache-ttl",
		envDurationOr(config.EnvCacheTTL, config.DefaultCacheTTL),
		"How long a cached entry stays valid before a plain fetch re-hits the network "+
			"(e.g. 720h = 30 days, 24h, 0 to always treat cache as stale). env: IBMDOCS_CACHE_TTL. "+
			"Use --refresh on fetch/fetch-file to force a re-fetch for a single invocation regardless of this value.")

	f.StringVar(&cfg.ProxyURL, "proxy",
		envOr(config.EnvProxy, ""),
		"HTTP/HTTPS proxy URL (e.g. http://proxy.corp:3128). "+
			"Overrides HTTP_PROXY/HTTPS_PROXY env vars. env: IBMDOCS_PROXY")

	f.StringVar(&cfg.TLSCABundle, "tls-ca-bundle",
		envOr(config.EnvTLSCABundle, ""),
		"Path to a PEM CA bundle to append to the system certificate pool. "+
			"Required when a TLS-inspecting proxy uses a private CA not already in /etc/ssl/certs/. "+
			"env: IBMDOCS_TLS_CA_BUNDLE")

	// Validate proxy/TLS flags at startup so the user gets a clear error
	// before any network call is attempted, and resolve the data folder.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := ibmdocs.ValidateTransportConfig(cfg.ProxyURL, cfg.TLSCABundle); err != nil {
			return err
		}
		if cfg.DataDir == "" && needsDataDir(cmd) {
			resolveDataDir(buildLogger(cfg.Verbose))
		}
		return nil
	}
}

// needsDataDir reports whether the command uses the data folder. Commands that
// do not (version, help, shell completion) must not touch the filesystem.
func needsDataDir(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "version", "help", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
			return false
		}
	}
	return true
}

// resolveDataDir fills cfg.DataDir from the environment/home fallbacks and
// logs every decision at debug level to avoid operational confusion.
func resolveDataDir(logger *slog.Logger) {
	r := config.ResolveDataDir()
	for _, n := range r.Notes {
		logger.Debug("data folder resolution", "note", n)
	}
	logger.Debug("data folder resolved", "dir", r.Dir, "source", r.Source)
	if r.Source == config.SourceCWD {
		logger.Debug("falling back to ./ibmdocs-data; set " + config.EnvDataFolder + " or --data to choose explicitly")
	}
	cfg.DataDir = r.Dir
}

// envOr returns the environment variable value or fallback.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envDurationOr returns the parsed environment variable duration or fallback
// if unset or invalid.
func envDurationOr(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
