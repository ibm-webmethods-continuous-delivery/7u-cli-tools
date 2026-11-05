// Package config holds the runtime Config struct shared across all commands.
package config

// Config is populated from global flags and passed into every command.
type Config struct {
	Verbose bool
	NoColor bool
	JSON    bool
}
