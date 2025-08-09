package tui

import (
	// "github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type Model struct {
	width int
	height int
	tasks []Task
}

type Task struct {
	title string
	status TaskStatus
	subtasks []Task
}

type TaskStatus int

const (
	Todo TaskStatus = iota
	Active
	Done
)

func NewModel() Model {
	return Model{
		tasks: initializeMockTasks(),
	}
}

func initializeMockTasks() []Task {
	tasks := []Task{
		{title: "his task is done", status: Done},
		{
			title: "This task is in progress",
			status: Active,
			subtasks: []Task{
				{title: "Subtask for active task", status: Todo},
			},
		},
		{title: "This task is waiting", status: Todo},
		{title: "Refactor input handling #ui", status: Active},
		{title: "Implement tag filter, this is a particularly long task to ensure multiline works correctly #feature", status: Todo},
		{title: "Initial scaffolding #setup", status: Done},
		{title: "Hook up keybindings #ux", status: Todo},
		{
			title: "Persist tasks to file #storage",
			status: Todo,
			subtasks: []Task{
				{
					title: "Implement save logic",
					status: Active,
					subtasks: []Task{
						{title: "Write helper function for JSON marshaling", status: Done},
						{title: "Write helper function for random stuff", status: Active},
					},
				},
			},
		},
		{title: "Add basic styling #ui", status: Done},
		{title: "Write unit tests #tests", status: Todo},
	}
	return tasks
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	title := lipgloss.NewStyle().
		Render("My dotdots")

	// Compute inner content width to enable wrapping within the padded container.
	// We have 2 chars padding on left and right (total = 4).
	innerWidth := m.width - 4
	if innerWidth < 0 {
		innerWidth = 0
	}

	var rows []string
	
	// Helper function to recursively render tasks and subtasks
	var renderTasks func(tasks []Task, indentLevel int)
	renderTasks = func(tasks []Task, indentLevel int) {
		for _, task := range tasks {
			rows = append(rows, m.renderRow(task, innerWidth, indentLevel))
			if len(task.subtasks) > 0 {
				renderTasks(task.subtasks, indentLevel+1)
			}
		}
	}
	
	renderTasks(m.tasks, 0)

	// Body: title + rows
	body := lipgloss.JoinVertical(lipgloss.Left, append([]string{title}, rows...)...)

	// Wrap the entire body to inner width (mainly affects the title line).
	wrapped := lipgloss.NewStyle().
		Width(innerWidth).
		MaxWidth(innerWidth).
		Render(body)

	// Padded container with fixed outer width.
	container := lipgloss.NewStyle().
		Padding(1, 2).
		Width(innerWidth + 4).
		MaxWidth(innerWidth + 4).
		Render(wrapped)

	return container
}

func (m Model) renderRow(task Task, width int, indentLevel int) string {
	// Styles: complete = greyed + strikethrough, active = green, todo = default
	completeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")). // terminal gray
		Strikethrough(true)
	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")) // terminal green
	todoStyle := lipgloss.NewStyle()   // regular/default color

	getStyledTaskText := func(t Task) string {
		var statusStr string
		switch t.status {
		case Done:
			statusStr = "@done"
		case Active:
			statusStr = "@active"
		case Todo:
			statusStr = ""
		}
		fullText := t.title + " " + statusStr
		switch t.status {
		case Done:
			return completeStyle.Render(fullText)
		case Active:
			return activeStyle.Render(fullText)
		default: // Todo
			return todoStyle.Render(fullText)
		}
	}

	// Create indentation string ((indentLevel-1)*2 spaces, then ╰ followed by a single space)
	indent := ""
	for i := 0; i < indentLevel-1; i++ {
		indent += "  "
	}
	if indentLevel > 0 {
		indent += "╰ "
	}

	var bullet string
	switch task.status {
	case Done:
		bullet = "◉"
	case Active:
		bullet = "◎"
	case Todo:
		bullet = "○"
	}

	// Add bullet style with width of 2 (bullet + space)
	bulletStyle := lipgloss.NewStyle().Width(2)
	bulletRendered := bulletStyle.Render(bullet + " ")

	// Text column width leaves space for bullet and indentation
	// For indentation: 0 levels = 0 chars, 1+ levels = indentLevel * 2 chars
	textColWidth := width - 2 - (indentLevel * 2)
	if textColWidth < 0 {
		textColWidth = 0
	}

	textStyle := lipgloss.NewStyle().Width(textColWidth)
	text := getStyledTaskText(task)
	textRendered := textStyle.Render(text)

	// Combine indent, bullet, and text
	return lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Render(indent), bulletRendered, textRendered)
}

// Ensure Model implements tea.Model
var _ tea.Model = (*Model)(nil)
