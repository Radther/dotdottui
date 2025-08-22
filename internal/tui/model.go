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

// getAdjacentTaskID returns the ID of the adjacent task in the given direction
// direction: -1 for previous, +1 for next
func (m Model) getAdjacentTaskID(direction int) string {
	ids := m.getAllTaskIDs()
	for i, id := range ids {
		if id == m.cursorID {
			newIndex := i + direction
			if newIndex >= 0 && newIndex < len(ids) {
				return ids[newIndex]
			}
			break
		}
	}
	return m.cursorID // Return current if at boundary
}

// getPreviousTaskID returns the ID of the previous task in traversal order
func (m Model) getPreviousTaskID() string {
	return m.getAdjacentTaskID(-1)
}

// getNextTaskID returns the ID of the next task in traversal order
func (m Model) getNextTaskID() string {
	return m.getAdjacentTaskID(1)
}

// findParentTask finds the parent task for a given task ID and returns the parent and index
// For top-level tasks, returns nil parent and the index in the top-level tasks slice
func (m *Model) findParentTask(taskID string) (*Task, int) {
	// Check if it's a top-level task first
	for i, task := range m.tasks {
		if task.id == taskID {
			return nil, i // No parent for top-level tasks
		}
	}
	
	// Helper function to recursively search for parent
	var search func(tasks *[]Task) (*Task, int)
	search = func(tasks *[]Task) (*Task, int) {
		for i := range *tasks {
			// Check if any subtask matches the target ID
			for j, subtask := range (*tasks)[i].subtasks {
				if subtask.id == taskID {
					return &(*tasks)[i], j
				}
			}
			// Recursively search deeper
			if len((*tasks)[i].subtasks) > 0 {
				if parent, index := search(&(*tasks)[i].subtasks); parent != nil {
					return parent, index
				}
			}
		}
		return nil, -1
	}
	
	return search(&m.tasks)
}

// getTaskContainer returns the slice containing the task based on parent info
func (m *Model) getTaskContainer(parent *Task) *[]Task {
	if parent == nil {
		return &m.tasks
	}
	return &parent.subtasks
}

// removeTaskFromSlice removes a task at the given index from a slice
func removeTaskFromSlice(slice *[]Task, index int) Task {
	task := (*slice)[index]
	copy((*slice)[index:], (*slice)[index+1:])
	*slice = (*slice)[:len(*slice)-1]
	return task
}

// insertTaskInSlice inserts a task at the given position in a slice
func insertTaskInSlice(slice *[]Task, index int, task Task) {
	*slice = append(*slice, Task{})
	copy((*slice)[index+1:], (*slice)[index:])
	(*slice)[index] = task
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
	parent, index := m.findParentTask(m.cursorID)
	if index <= 0 {
		return // Can't move up if not found or already first
	}
	
	container := m.getTaskContainer(parent)
	// Swap with the previous task
	(*container)[index], (*container)[index-1] = (*container)[index-1], (*container)[index]
}

// moveTaskDown moves a task down within its parent container
func (m *Model) moveTaskDown() {
	parent, index := m.findParentTask(m.cursorID)
	if index < 0 {
		return // Can't move down if not found
	}
	
	container := m.getTaskContainer(parent)
	if index >= len(*container)-1 {
		return // Can't move down if already last
	}
	
	// Swap with the next task
	(*container)[index], (*container)[index+1] = (*container)[index+1], (*container)[index]
}

// unindentTask moves a task out of its parent (decrease indentation)
func (m *Model) unindentTask() {
	parent, index := m.findParentTask(m.cursorID)
	if parent == nil {
		return // Can't unindent top-level tasks
	}
	
	// Remove task from current location (parent's subtasks)
	task := removeTaskFromSlice(&parent.subtasks, index)
	
	// Find where to insert the task (after its former parent)
	grandparent, parentIndex := m.findParentTask(parent.id)
	container := m.getTaskContainer(grandparent)
	
	// Insert task after its former parent
	insertTaskInSlice(container, parentIndex+1, task)
}

// indentTask moves a task into the previous sibling (increase indentation)
func (m *Model) indentTask() {
	parent, index := m.findParentTask(m.cursorID)
	if index <= 0 {
		return // Can't indent if not found or first task
	}
	
	container := m.getTaskContainer(parent)
	// Get the previous sibling (which will become the parent)
	prevSibling := &(*container)[index-1]
	
	// Remove task from current location
	task := removeTaskFromSlice(container, index)
	
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
	// Task status styles
	styleMap := map[TaskStatus]lipgloss.Style{
		Done:   lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Strikethrough(true),
		Active: lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		Todo:   lipgloss.NewStyle(),
	}
	
	// Task status bullets
	bulletMap := map[TaskStatus]string{
		Done:   "◉",
		Active: "◎",
		Todo:   "○",
	}

	// Create indentation string
	indent := ""
	for i := 0; i < indentLevel-1; i++ {
		indent += "  "
	}
	if indentLevel > 0 {
		indent += "╰ "
	}

	// Create styled components
	bulletRendered := lipgloss.NewStyle().Width(2).Render(bulletMap[task.status] + " ")
	textColWidth := width - 2 - 2 - (indentLevel * 2)
	if textColWidth < 0 {
		textColWidth = 0
	}

	// Handle editing mode
	if isEditing && isSelected {
		textRendered := lipgloss.NewStyle().Width(textColWidth).Render(m.textInput.View())
		cursorRendered := lipgloss.NewStyle().Width(2).Render("> ")
		return lipgloss.JoinHorizontal(lipgloss.Top, cursorRendered, lipgloss.NewStyle().Render(indent), bulletRendered, textRendered)
	}

	// Normal mode
	text := styleMap[task.status].Render(task.title)
	textRendered := lipgloss.NewStyle().Width(textColWidth).Render(text)
	cursorSymbol := " "
	if isSelected {
		cursorSymbol = ">"
	}
	cursorRendered := lipgloss.NewStyle().Width(2).Render(cursorSymbol + " ")
	
	return lipgloss.JoinHorizontal(lipgloss.Top, cursorRendered, lipgloss.NewStyle().Render(indent), bulletRendered, textRendered)
}

// Ensure Model implements tea.Model
var _ tea.Model = (*Model)(nil)
