package tui

import (
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/google/uuid"
)

type Model struct {
	width     int
	height    int
	tasks     []Task
	cursorID  string
	editing   bool
	textInput textinput.Model
}

type Task struct {
	id       string
	title    string
	status   TaskStatus
	subtasks []Task
}

type TaskStatus int

const (
	Todo TaskStatus = iota
	Active
	Done
)

// NewTask creates a new task with auto-generated UUID
func NewTask(title string, status TaskStatus, subtasks ...Task) Task {
	return Task{
		id:       uuid.New().String(),
		title:    title,
		status:   status,
		subtasks: subtasks,
	}
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Task title"
	ti.Focus()
	
	tasks := InitializeMockTasks()
	var cursorID string
	if len(tasks) > 0 {
		cursorID = tasks[0].id
	}
	
	return Model{
		tasks:     tasks,
		cursorID:  cursorID,
		editing:   false,
		textInput: ti,
	}
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
				m.editTaskTitle(m.cursorID, m.textInput.Value())
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
				m.cursorID = m.getPreviousTaskID()
			case "down", "j":
				m.cursorID = m.getNextTaskID()
			case "ctrl+up":
				// Move task up within its parent
				m.moveTaskUp()
			case "ctrl+down":
				// Move task down within its parent
				m.moveTaskDown()
			case "ctrl+left":
				// Unindent task (move out of parent)
				m.unindentTask()
			case "ctrl+right":
				// Indent task (move into previous sibling)
				m.indentTask()
			case "enter":
				// Toggle editing mode
				m.editing = true
				// Set the text input value to the current task title
				task := m.getCurrentTask()
				if task != nil {
					m.textInput.SetValue(task.title)
				}
				m.textInput.Focus()
				return m, nil
			}
		}
	}
	return m, nil
}

// findTaskByID finds a task by its UUID and returns it
func (m Model) findTaskByID(id string) *Task {
	var found *Task
	
	// Helper function to recursively traverse tasks
	var traverse func(tasks []Task)
	traverse = func(tasks []Task) {
		for i := range tasks {
			if found != nil {
				return
			}
			if tasks[i].id == id {
				found = &tasks[i]
				return
			}
			if len(tasks[i].subtasks) > 0 {
				traverse(tasks[i].subtasks)
			}
		}
	}
	
	traverse(m.tasks)
	return found
}

// getCurrentTask returns the currently selected task
func (m Model) getCurrentTask() *Task {
	return m.findTaskByID(m.cursorID)
}

// getAllTaskIDs returns all task IDs in traversal order
func (m Model) getAllTaskIDs() []string {
	var ids []string
	
	// Helper function to recursively traverse tasks
	var traverse func(tasks []Task)
	traverse = func(tasks []Task) {
		for _, task := range tasks {
			ids = append(ids, task.id)
			if len(task.subtasks) > 0 {
				traverse(task.subtasks)
			}
		}
	}
	
	traverse(m.tasks)
	return ids
}

// getPreviousTaskID returns the ID of the previous task in traversal order
func (m Model) getPreviousTaskID() string {
	ids := m.getAllTaskIDs()
	for i, id := range ids {
		if id == m.cursorID && i > 0 {
			return ids[i-1]
		}
	}
	return m.cursorID // Return current if at beginning
}

// getNextTaskID returns the ID of the next task in traversal order
func (m Model) getNextTaskID() string {
	ids := m.getAllTaskIDs()
	for i, id := range ids {
		if id == m.cursorID && i < len(ids)-1 {
			return ids[i+1]
		}
	}
	return m.cursorID // Return current if at end
}

// findTaskContainer finds the container (parent task slice) and index for a given task ID
func (m Model) findTaskContainer(taskID string) (*[]Task, int) {
	// Check top-level tasks first
	for i, task := range m.tasks {
		if task.id == taskID {
			return &m.tasks, i
		}
	}
	
	// Helper function to recursively search in subtasks
	var search func(tasks *[]Task) (*[]Task, int)
	search = func(tasks *[]Task) (*[]Task, int) {
		for i := range *tasks {
			for j, subtask := range (*tasks)[i].subtasks {
				if subtask.id == taskID {
					return &(*tasks)[i].subtasks, j
				}
			}
			// Recursively search deeper
			if len((*tasks)[i].subtasks) > 0 {
				if container, index := search(&(*tasks)[i].subtasks); container != nil {
					return container, index
				}
			}
		}
		return nil, -1
	}
	
	return search(&m.tasks)
}

func (m *Model) editTaskTitle(taskID string, newTitle string) {
	found := false
	
	// Helper function to recursively traverse and modify tasks
	var traverse func(tasks *[]Task)
	traverse = func(tasks *[]Task) {
		for i := range *tasks {
			if found {
				return
			}
			if (*tasks)[i].id == taskID {
				(*tasks)[i].title = newTitle
				found = true
				return
			}
			if len((*tasks)[i].subtasks) > 0 {
				traverse(&(*tasks)[i].subtasks)
			}
		}
	}
	
	traverse(&m.tasks)
}

// moveTaskUp moves a task up within its parent container
func (m *Model) moveTaskUp() {
	container, index := m.findTaskContainer(m.cursorID)
	if container == nil || index <= 0 {
		return // Can't move up if not found or already first
	}
	
	// Swap with the previous task
	(*container)[index], (*container)[index-1] = (*container)[index-1], (*container)[index]
}

// moveTaskDown moves a task down within its parent container
func (m *Model) moveTaskDown() {
	container, index := m.findTaskContainer(m.cursorID)
	if container == nil || index >= len(*container)-1 {
		return // Can't move down if not found or already last
	}
	
	// Swap with the next task
	(*container)[index], (*container)[index+1] = (*container)[index+1], (*container)[index]
}

// unindentTask moves a task out of its parent (decrease indentation)
func (m *Model) unindentTask() {
	container, index := m.findTaskContainer(m.cursorID)
	if container == nil {
		return
	}
	
	// Find parent container (only works if current task is not top-level)
	var parentContainer *[]Task
	var parentIndex int
	found := false
	
	// Search for the parent of the current container
	var search func(tasks *[]Task, targetContainer *[]Task) bool
	search = func(tasks *[]Task, targetContainer *[]Task) bool {
		for i := range *tasks {
			if &(*tasks)[i].subtasks == targetContainer {
				parentContainer = tasks
				parentIndex = i
				return true
			}
			if len((*tasks)[i].subtasks) > 0 {
				if search(&(*tasks)[i].subtasks, targetContainer) {
					return true
				}
			}
		}
		return false
	}
	
	// Check if this is a top-level task (can't unindent)
	if container == &m.tasks {
		return
	}
	
	found = search(&m.tasks, container)
	if !found {
		return
	}
	
	// Remove task from current location
	task := (*container)[index]
	copy((*container)[index:], (*container)[index+1:])
	*container = (*container)[:len(*container)-1]
	
	// Insert task after its parent
	insertPos := parentIndex + 1
	*parentContainer = append(*parentContainer, Task{})
	copy((*parentContainer)[insertPos+1:], (*parentContainer)[insertPos:])
	(*parentContainer)[insertPos] = task
}

// indentTask moves a task into the previous sibling (increase indentation)
func (m *Model) indentTask() {
	container, index := m.findTaskContainer(m.cursorID)
	if container == nil || index <= 0 {
		return // Can't indent if not found or first task
	}
	
	// Get the previous sibling (which will become the parent)
	prevSibling := &(*container)[index-1]
	
	// Remove task from current location
	task := (*container)[index]
	copy((*container)[index:], (*container)[index+1:])
	*container = (*container)[:len(*container)-1]
	
	// Add task as subtask of previous sibling
	prevSibling.subtasks = append(prevSibling.subtasks, task)
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
	var renderTasks func(tasks []Task, indentLevel int)
	renderTasks = func(tasks []Task, indentLevel int) {
		for _, task := range tasks {
			isSelected := task.id == m.cursorID
			rows = append(rows, m.renderRow(task, innerWidth, indentLevel, isSelected, m.editing))
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
