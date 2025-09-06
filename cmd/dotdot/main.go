package main

import (
	"dotdot/internal/cli"
	"dotdot/internal/storage"
	"dotdot/internal/tui"
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
)

func main() {
	cmd, err := cli.ParseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	switch cmd.Action {
	case "open":
		runTUI(cmd.FilePath)
	case "list":
		listTasks(cmd)
	case "delete":
		deleteTasks(cmd)
	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s\n", cmd.Action)
		os.Exit(1)
	}
}

func runTUI(filePath string) {
	model := tui.NewModelWithFile(filePath)

	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		log.Fatal(err)
	}
}

func listTasks(cmd *cli.Command) {
	var taskLists []string
	var err error
	var location, emptyMsg string

	if cmd.Local {
		taskLists, err = storage.ListLocalTasks()
		location = "Local"
		emptyMsg = "No local task lists found in current directory"
	} else {
		taskLists, err = storage.ListGlobalTasks()
		location = "Global"
		emptyMsg = "No global task lists found"
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing %s tasks: %v\n", strings.ToLower(location), err)
		os.Exit(1)
	}

	if len(taskLists) == 0 {
		fmt.Println(emptyMsg)
	} else {
		fmt.Printf("%s task lists:\n", location)
		for _, name := range taskLists {
			if cmd.Local {
				fmt.Printf("  %s.dot\n", name)
			} else {
				fmt.Printf("  %s\n", name)
			}
		}
	}
}

func deleteTasks(cmd *cli.Command) {
	if !storage.FileExists(cmd.FilePath) {
		fmt.Fprintf(os.Stderr, "Task list file does not exist: %s\n", cmd.FilePath)
		os.Exit(1)
	}

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete '%s'? (y/N): ", cmd.FilePath)
	var response string
	if _, err := fmt.Scanln(&response); err != nil || (response != "y" && response != "Y" && response != "yes" && response != "Yes") {
		fmt.Println("Deletion cancelled")
		return
	}

	if err := storage.DeleteTaskList(cmd.FilePath); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting task list: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully deleted task list: %s\n", cmd.FilePath)
}
