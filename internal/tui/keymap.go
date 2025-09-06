package tui

import "github.com/charmbracelet/bubbles/v2/key"

// KeyMap defines all keyboard shortcuts for the application
type KeyMap struct {
	// Navigation
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding

	// Task creation
	NewTaskBelow    key.Binding
	NewSubtask      key.Binding
	NewTaskInParent key.Binding

	// Task management
	MoveUp       key.Binding
	MoveDown     key.Binding
	IndentTask   key.Binding
	UnindentTask key.Binding

	// Edit mode
	EditTask               key.Binding
	Confirm                key.Binding
	Cancel                 key.Binding
	NewTaskBelowFromEdit   key.Binding
	NewTaskInParentFromEdit key.Binding

	// Undo/Redo
	Undo key.Binding
	Redo key.Binding

	// Clipboard
	Copy           key.Binding
	Paste          key.Binding
	PasteAsSubtask key.Binding

	// General
	Help key.Binding
	Quit key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Left, k.Right, k.NewTaskBelow, k.EditTask, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		// Navigation
		{k.Up, k.Down, k.Left, k.Right},
		// Task Operations
		{k.NewTaskBelow, k.NewSubtask, k.NewTaskInParent, k.EditTask},
		// Task Management
		{k.MoveUp, k.MoveDown, k.IndentTask, k.UnindentTask},
		// Edit & Actions
		{k.Undo, k.Redo, k.Copy, k.Paste, k.PasteAsSubtask},
		// Edit Mode Actions
		{k.NewTaskBelowFromEdit, k.NewTaskInParentFromEdit},
		// General
		{k.Help, k.Quit},
	}
}

// DefaultKeyMap returns a default set of keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("↓/j", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("←/h", "status back"),
		),
		Right: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("→/l", "status forward"),
		),

		// Task creation
		NewTaskBelow: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new task below"),
		),
		NewSubtask: key.NewBinding(
			key.WithKeys("N", "shift+enter"),
			key.WithHelp("N/shift+↵", "new subtask"),
		),
		NewTaskInParent: key.NewBinding(
			key.WithKeys("ctrl+n", "ctrl+enter"),
			key.WithHelp("ctrl+n/ctrl+↵", "new task in parent"),
		),

		// Task management
		MoveUp: key.NewBinding(
			key.WithKeys("ctrl+k", "ctrl+up"),
			key.WithHelp("ctrl+↑/k", "move task up"),
		),
		MoveDown: key.NewBinding(
			key.WithKeys("ctrl+j", "ctrl+down"),
			key.WithHelp("ctrl+↓/j", "move task down"),
		),
		IndentTask: key.NewBinding(
			key.WithKeys("ctrl+l", "ctrl+right"),
			key.WithHelp("ctrl+→/l", "indent task"),
		),
		UnindentTask: key.NewBinding(
			key.WithKeys("ctrl+h", "ctrl+left"),
			key.WithHelp("ctrl+←/h", "unindent task"),
		),

		// Edit mode
		EditTask: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "edit task"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		NewTaskBelowFromEdit: key.NewBinding(
			key.WithKeys("shift+enter"),
			key.WithHelp("shift+↵", "save & new task below"),
		),
		NewTaskInParentFromEdit: key.NewBinding(
			key.WithKeys("ctrl+enter"),
			key.WithHelp("ctrl+↵", "save & new task in parent"),
		),

		// Undo/Redo
		Undo: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "undo"),
		),
		Redo: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "redo"),
		),

		// Clipboard
		Copy: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy task"),
		),
		Paste: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "paste task"),
		),
		PasteAsSubtask: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "paste as subtask"),
		),

		// General
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}
