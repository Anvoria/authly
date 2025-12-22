package cli

import (
	"flag"
	"fmt"
	"os"
)

// Command represents a CLI command
type Command interface {
	Name() string
	Description() string
	Run(args []string) error
}

// Registry holds the registered commands
type Registry struct {
	commands map[string]Command
}

// NewRegistry creates a new command registry
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]Command),
	}
}

// Register adds a command to the registry
func (r *Registry) Register(cmd Command) {
	r.commands[cmd.Name()] = cmd
}

// Run executes a command based on the first argument
func (r *Registry) Run(args []string) error {
	if len(args) < 1 {
		r.printUsage()
		return fmt.Errorf("command required")
	}

	cmdName := args[0]
	cmd, ok := r.commands[cmdName]
	if !ok {
		r.printUsage()
		return fmt.Errorf("unknown command: %s", cmdName)
	}

	return cmd.Run(args[1:])
}

func (r *Registry) printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: authly-cli <command> [args]\n\n")
	fmt.Fprintf(os.Stderr, "Available commands:\n")
	for name, cmd := range r.commands {
		fmt.Fprintf(os.Stderr, "  %-15s %s\n", name, cmd.Description())
	}
}

// Helper to standardise sub-command parsing
func ParseFlags(name string, args []string, setup func(*flag.FlagSet)) (*flag.FlagSet, error) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	setup(fs)
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return fs, nil
}
