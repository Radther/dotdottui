package tui

import "github.com/charmbracelet/lipgloss/v2"

// Color constants by semantic use
const (
	CursorColor      = "1" // Red - cursor and selection indicator
	ActiveTaskColor  = "2" // Green - active tasks
	DimmedColor      = "8" // Gray - dimmed/disabled elements  
	ErrorBgColor     = "0" // Black - error message background
	ErrorTextColor   = "1" // Red - error text
)

// UI spacing constants
const (
	CursorWidth  = 2
	BulletWidth  = 2
	IndentWidth  = 2
	PaddingLeft  = 2
	PaddingRight = 2
	TotalPadding = PaddingLeft + PaddingRight
)

// Pre-defined styles for consistent UI elements
var (
	// Error message styling
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ErrorTextColor)).
			Background(lipgloss.Color(ErrorBgColor)).
			Padding(0, 1).
			Margin(1, 0)

	// Help text styling
	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(DimmedColor)).
			Italic(true)

	// Task status styles
	TaskDoneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(DimmedColor)).
			Strikethrough(true)

	TaskActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ActiveTaskColor))

	TaskTodoStyle = lipgloss.NewStyle()

	// Bullet styling
	BulletStyle = lipgloss.NewStyle().Width(BulletWidth)

	BulletDimmedStyle = lipgloss.NewStyle().
				Width(BulletWidth).
				Foreground(lipgloss.Color(DimmedColor))

	// Cursor styling
	CursorStyle = lipgloss.NewStyle().Width(CursorWidth)

	CursorSelectedStyle = lipgloss.NewStyle().
				Width(CursorWidth).
				Foreground(lipgloss.Color(CursorColor))

	CursorDimmedStyle = lipgloss.NewStyle().
				Width(CursorWidth).
				Foreground(lipgloss.Color(DimmedColor))
)

// Task status bullet symbols
var BulletSymbols = map[TaskStatus]string{
	Done:   "◉",
	Active: "◎", 
	Todo:   "○",
}

// GetTaskStyle returns the appropriate style for a task based on its status
func GetTaskStyle(status TaskStatus) lipgloss.Style {
	switch status {
	case Done:
		return TaskDoneStyle
	case Active:
		return TaskActiveStyle
	case Todo:
		return TaskTodoStyle
	default:
		return TaskTodoStyle
	}
}