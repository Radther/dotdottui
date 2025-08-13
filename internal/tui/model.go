package tui

import (
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type Model struct {
	width int
	height int
	tasks []Task
	cursor int
	editing bool
	textInput textinput.Model
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
	ti := textinput.New()
	ti.Placeholder = "Task title"
	ti.Focus()
	
	return Model{
		tasks: initializeMockTasks(),
		cursor: 0,
		editing: false,
		textInput: ti,
	}
}

func initializeMockTasks() []Task {
	tasks := []Task{
		{title: "This task is done", status: Done},
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
		if m.editing {
			// Handle text input updates when in editing mode
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			
			switch msg.String() {
			case "enter":
				// Save the edited text and exit editing mode
				m.editTaskTitle(m.cursor, m.textInput.Value())
				m.editing = false
				m.textInput.Blur()
				return m, cmd
			case "esc":
				// Exit editing mode without saving
				m.editing = false
				m.textInput.Blur()
				return m, cmd
			}
			
			return m, cmd
		} else {
			// Handle normal mode updates
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				// Count total tasks to determine bounds
				totalTasks := m.countTasks()
				if m.cursor < totalTasks-1 {
					m.cursor++
				}
			case "enter":
				// Toggle editing mode
				m.editing = true
				// Set the text input value to the current task title
				task := m.getTaskAtPosition(m.cursor)
				m.textInput.SetValue(task.title)
				m.textInput.Focus()
				return m, nil
			}
		}
	}
	return m, nil
}

func (m Model) countTasks() int {
	count := 0
	for _, task := range m.tasks {
		count++
		count += m.countSubTasks(task)
	}
	return count
}

func (m Model) countSubTasks(task Task) int {
	count := 0
	for _, subtask := range task.subtasks {
		count++
		count += m.countSubTasks(subtask)
	}
	return count
}

func (m Model) getTaskAtPosition(pos int) Task {
	index := 0
	var foundTask Task
	found := false
	
	// Helper function to recursively traverse tasks
	var traverse func(tasks []Task)
	traverse = func(tasks []Task) {
		for i := range tasks {
			if found {
				return
			}
			if index == pos {
				foundTask = tasks[i]
				found = true
				return
			}
			index++
			if len(tasks[i].subtasks) > 0 {
				traverse(tasks[i].subtasks)
			}
		}
	}
	
	traverse(m.tasks)
	return foundTask
}

func (m *Model) editTaskTitle(pos int, newTitle string) {
	index := 0
	found := false
	
	// Helper function to recursively traverse and modify tasks
	var traverse func(tasks *[]Task)
	traverse = func(tasks *[]Task) {
		for i := range *tasks {
			if found {
				return
			}
			if index == pos {
				(*tasks)[i].title = newTitle
				found = true
				return
			}
			index++
			if len((*tasks)[i].subtasks) > 0 {
				traverse(&(*tasks)[i].subtasks)
			}
		}
	}
	
	traverse(&m.tasks)
}

func (m Model) View() string {
	title := lipgloss.NewStyle().
		Render("Totally real task list")

	// Compute inner content width to enable wrapping within the padded container.
	// We have 2 chars padding on left and right (total = 4).
	innerWidth := m.width - 4
	if innerWidth < 0 {
		innerWidth = 0
	}

	var rows []string
	
	// Helper function to recursively render tasks and subtasks
	var renderTasks func(tasks []Task, indentLevel int, index *int)
	renderTasks = func(tasks []Task, indentLevel int, index *int) {
		for _, task := range tasks {
			isSelected := *index == m.cursor
			rows = append(rows, m.renderRow(task, innerWidth, indentLevel, isSelected, m.editing))
			(*index)++
			if len(task.subtasks) > 0 {
				renderTasks(task.subtasks, indentLevel+1, index)
			}
		}
	}
	
	index := 0
	renderTasks(m.tasks, 0, &index)

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

func (m Model) renderRow(task Task, width int, indentLevel int, isSelected bool, isEditing bool) string {
	// Styles: complete = greyed + strikethrough, active = green, todo = default
	completeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")). // terminal gray
		Strikethrough(true)
	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")) // terminal green
	todoStyle := lipgloss.NewStyle()   // regular/default color

	getStyledTaskText := func(t Task) string {
		switch t.status {
		case Done:
			return completeStyle.Render(t.title)
		case Active:
			return activeStyle.Render(t.title)
		default: // Todo
			return todoStyle.Render(t.title)
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

	// Text column width leaves space for cursor, bullet and indentation
	// For indentation: 0 levels = 0 chars, 1+ levels = indentLevel * 2 chars
	// Cursor takes 2 chars, bullet takes 2 chars
	textColWidth := width - 2 - 2 - (indentLevel * 2)
	if textColWidth < 0 {
		textColWidth = 0
	}

	// When editing the selected task, show the text input instead of the task title
	if isEditing && isSelected {
		textStyle := lipgloss.NewStyle().Width(textColWidth)
		textRendered := textStyle.Render(m.textInput.View())
		cursorStyle := lipgloss.NewStyle().Width(2)
		cursorRendered := cursorStyle.Render("> ")
		return lipgloss.JoinHorizontal(lipgloss.Top, cursorRendered, lipgloss.NewStyle().Render(indent), bulletRendered, textRendered)
	}

	textStyle := lipgloss.NewStyle().Width(textColWidth)
	text := getStyledTaskText(task)
	textRendered := textStyle.Render(text)

	// Combine cursor, indent, bullet, and text
	cursorSymbol := " "
	if isSelected {
		cursorSymbol = ">"
	}
	
	cursorStyle := lipgloss.NewStyle().Width(2)
	cursorRendered := cursorStyle.Render(cursorSymbol + " ")
	
	return lipgloss.JoinHorizontal(lipgloss.Top, cursorRendered, lipgloss.NewStyle().Render(indent), bulletRendered, textRendered)
}

// Ensure Model implements tea.Model
var _ tea.Model = (*Model)(nil)
