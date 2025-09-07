package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dotdot/internal/storage"
)

// Command represents the parsed command and its arguments
type Command struct {
	Action   string // "open", "list", "delete"
	Name     string // task list name for global lists
	Local    bool   // --local flag
	File     string // --file flag value
	FilePath string // resolved file path to use
}

// ParseArgs parses command line arguments and returns a Command
func ParseArgs() (*Command, error) {
	// Define flags
	var (
		local = flag.Bool("local", false, "Use local task list in current directory")
		file  = flag.String("file", "", "Use specific file path")
		help  = flag.Bool("help", false, "Show help information")
	)

	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] [command] [name]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  open [name]    Open a task list (default command)\n")
		fmt.Fprintf(os.Stderr, "  list           List available task lists\n")
		fmt.Fprintf(os.Stderr, "  delete [name]  Delete a task list\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                        # Open default tasks.dot in current directory\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s open work              # Open global 'work' task list\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --local open mytasks   # Open mytasks.dot in current directory\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --file ~/tasks.dot open # Open specific file\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s list                   # List global task lists\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --local list           # List local .dot files\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s delete work            # Delete global 'work' task list\n", os.Args[0])
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	args := flag.Args()

	cmd := &Command{
		Local: *local,
		File:  *file,
	}

	// Parse command and name from remaining args
	switch len(args) {
	case 0:
		// No arguments: default to opening local tasks.dot
		cmd.Action = "open"
		cmd.Name = "tasks"
		cmd.Local = true
	case 1:
		// One argument: could be a command or a name
		if args[0] == "list" {
			cmd.Action = "list"
		} else if args[0] == "delete" {
			return nil, fmt.Errorf("delete command requires a name")
		} else {
			// Assume it's a task list name
			cmd.Action = "open"
			cmd.Name = strings.TrimSuffix(args[0], ".dot")
		}
	case 2:
		// Two arguments: command and name
		cmd.Action = args[0]
		cmd.Name = strings.TrimSuffix(args[1], ".dot")

		if cmd.Action != "open" && cmd.Action != "delete" {
			return nil, fmt.Errorf("invalid command: %s", cmd.Action)
		}
	default:
		return nil, fmt.Errorf("too many arguments")
	}

	// Validate flag combinations
	if *local && *file != "" {
		return nil, fmt.Errorf("cannot use both --local and --file flags")
	}

	if cmd.Action == "list" && cmd.Name != "" {
		return nil, fmt.Errorf("list command does not accept a name argument")
	}

	// Resolve file path
	var err error
	cmd.FilePath, err = cmd.resolveFilePath()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

// resolveFilePath determines the actual file path to use based on the command flags
func (c *Command) resolveFilePath() (string, error) {
	switch {
	case c.File != "":
		// Explicit file path
		return c.File, nil
	case c.Local:
		// Local file in current directory
		if c.Name == "" {
			c.Name = "tasks"
		}
		return c.Name + ".dot", nil
	default:
		// Global task list
		if c.Name == "" {
			c.Name = "tasks"
		}

		configDir, err := storage.GetConfigDir()
		if err != nil {
			return "", fmt.Errorf("failed to get config directory: %w", err)
		}

		tasksDir := filepath.Join(configDir, "dotdot", "tasks")
		return filepath.Join(tasksDir, c.Name+".dot"), nil
	}
}

// IsGlobal returns true if this command operates on global task lists
func (c *Command) IsGlobal() bool {
	return !c.Local && c.File == ""
}

// IsLocal returns true if this command operates on local task lists
func (c *Command) IsLocal() bool {
	return c.Local
}

// IsFile returns true if this command operates on a specific file
func (c *Command) IsFile() bool {
	return c.File != ""
}
