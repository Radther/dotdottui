package main

import (
	"dotdot/internal/cli"
	"dotdot/internal/storage"
	"dotdot/internal/tui"
	"fmt"
	"log"
	"os"

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
	
	if cmd.Local {
		taskLists, err = storage.ListLocalTasks()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing local tasks: %v\n", err)
			os.Exit(1)
		}
		
		if len(taskLists) == 0 {
			fmt.Println("No local task lists found in current directory")
		} else {
			fmt.Println("Local task lists:")
			for _, name := range taskLists {
				fmt.Printf("  %s.dot\n", name)
			}
		}
	} else {
		taskLists, err = storage.ListGlobalTasks()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing global tasks: %v\n", err)
			os.Exit(1)
		}
		
		if len(taskLists) == 0 {
			fmt.Println("No global task lists found")
		} else {
			fmt.Println("Global task lists:")
			for _, name := range taskLists {
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
