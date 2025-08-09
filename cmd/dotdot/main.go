package main

import (
	"dotdot/internal/tui"
	"log"

	tea "github.com/charmbracelet/bubbletea/v2"
)

func main() {
	if _, err := tea.NewProgram(tui.NewModel(), tea.WithAltScreen()).Run(); err != nil {
		log.Fatal(err)
	}
}
