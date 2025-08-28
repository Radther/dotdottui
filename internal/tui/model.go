package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	
	"dotdot/internal/storage"
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/google/uuid"
)

type Model struct {
	width        int
	height       int
	tasks        []Task
	cursorID     string
	previousID   string
	editing      bool
	textInput    textinput.Model
	filePath     string  // Path to the current task file
	autoSave     bool    // Enable auto-save after operations
	lastError    string  // Last error message to display
	showError    bool    // Whether to show the error message
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

const (
	// UI spacing constants
	CursorWidth     = 2
	BulletWidth     = 2
	IndentWidth     = 2
	PaddingLeft     = 2
	PaddingRight    = 2
	TotalPadding    = PaddingLeft + PaddingRight
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

// NewTaskWithID creates a new task with a specific ID (used for loading from storage)
func NewTaskWithID(id, title string, status TaskStatus, subtasks ...Task) Task {
	return Task{
		id:       id,
		title:    title,
		status:   status,
		subtasks: subtasks,
	}
}

// Accessor methods for Task
func (t Task) ID() string {
	return t.id
}

func (t Task) Title() string {
	return t.title
}

func (t Task) Status() TaskStatus {
	return t.status
}

func (t Task) Subtasks() []Task {
	return t.subtasks
}

func NewModel() Model {
	return NewModelWithFile("")
}

func NewModelWithFile(filePath string) Model {
	ti := textinput.New()
	ti.Placeholder = "Task title"
	ti.Prompt = ""
	ti.Focus()
	
	var tasks []Task
	var cursorID string
	
	var loadError string
	
	// Load tasks from file if specified, otherwise use mock data
	if filePath != "" {
		if loadedTasks, err := loadTasksFromFile(filePath); err == nil {
			tasks = loadedTasks
		} else {
			// On error, start with empty task list and show error
			tasks = []Task{}
			loadError = "Failed to load tasks: " + err.Error()
		}
	} else {
		tasks = InitializeMockTasks()
	}
	
	if len(tasks) > 0 {
		cursorID = tasks[0].id
	}
	
	return Model{
		tasks:      tasks,
		cursorID:   cursorID,
		previousID: "",
		editing:    false,
		textInput:  ti,
		filePath:   filePath,
		autoSave:   filePath != "", // Enable auto-save when file path is provided
		lastError:  loadError,
		showError:  loadError != "",
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
			return m.handleEditingMode(msg)
		} else {
			return m.handleNormalMode(msg)
		}
	}
	return m, nil
}

func (m Model) handleEditingMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	
	switch msg.String() {
	case "enter":
		m.editTaskTitle(m.cursorID, m.textInput.Value())
		m.editing = false
		m.textInput.Blur()
		return m, cmd
	case "esc":
		// If the task title is empty, delete the task
		currentTask := m.getCurrentTask()
		if currentTask != nil && currentTask.title == "" {
			m.deleteCurrentTask()
		}
		m.editing = false
		m.textInput.Blur()
		return m, cmd
	}
	
	return m, cmd
}

func (m Model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		// Clear error messages on ESC
		if m.showError {
			m.clearError()
			return m, nil
		}
		// If no error to clear, do nothing
	case "up", "k":
		m.cursorID = m.getPreviousTaskID()
	case "down", "j":
		m.cursorID = m.getNextTaskID()
	case "left", "h":
		m.changeTaskStatusBackward()
	case "right", "l":
		m.changeTaskStatusForward()
	case "ctrl+up", "ctrl+k":
		m.moveTaskUp()
	case "ctrl+down", "ctrl+j":
		m.moveTaskDown()
	case "ctrl+left", "ctrl+h":
		m.unindentTask()
	case "ctrl+right", "ctrl+l":
		m.indentTask()
	case "n":
		m.previousID = m.cursorID
		newTaskID := m.createNewTaskBelow()
		if newTaskID != "" {
			m.cursorID = newTaskID
			m.editing = true
			m.textInput.SetValue("")
			m.textInput.Focus()
		}
		return m, nil
	case "N":
		m.previousID = m.cursorID
		newTaskID := m.createNewSubtask()
		if newTaskID != "" {
			m.cursorID = newTaskID
			m.editing = true
			m.textInput.SetValue("")
			m.textInput.Focus()
		}
		return m, nil
	case "enter":
		m.editing = true
		task := m.getCurrentTask()
		if task != nil {
			m.textInput.SetValue(task.title)
		}
		m.textInput.Focus()
		return m, nil
	}
	return m, nil
}

// traverseTasks executes a function for each task in the tree
func (m Model) traverseTasks(fn func(*Task) bool) {
	var traverse func(tasks []Task) bool
	traverse = func(tasks []Task) bool {
		for i := range tasks {
			if fn(&tasks[i]) {
				return true
			}
			if len(tasks[i].subtasks) > 0 {
				if traverse(tasks[i].subtasks) {
					return true
				}
			}
		}
		return false
	}
	traverse(m.tasks)
}

// findTaskByID finds a task by its UUID and returns it
func (m Model) findTaskByID(id string) *Task {
	var found *Task
	m.traverseTasks(func(task *Task) bool {
		if task.id == id {
			found = task
			return true
		}
		return false
	})
	return found
}

// getCurrentTask returns the currently selected task
func (m Model) getCurrentTask() *Task {
	return m.findTaskByID(m.cursorID)
}

// getAllTaskIDs returns all task IDs in traversal order
func (m Model) getAllTaskIDs() []string {
	var ids []string
	m.traverseTasks(func(task *Task) bool {
		ids = append(ids, task.id)
		return false
	})
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
	m.traverseTasks(func(task *Task) bool {
		if task.id == taskID {
			task.title = newTitle
			return true
		}
		return false
	})
	m.autoSaveIfEnabled()
}

// changeTaskStatusForward advances task status: Todo -> Active -> Done
func (m *Model) changeTaskStatusForward() {
	m.traverseTasks(func(task *Task) bool {
		if task.id == m.cursorID {
			switch task.status {
			case Todo:
				task.status = Active
			case Active:
				task.status = Done
			case Done:
				// Already at max status, no change
			}
			return true
		}
		return false
	})
	m.autoSaveIfEnabled()
}

// changeTaskStatusBackward reverses task status: Done -> Active -> Todo
func (m *Model) changeTaskStatusBackward() {
	m.traverseTasks(func(task *Task) bool {
		if task.id == m.cursorID {
			switch task.status {
			case Done:
				task.status = Active
			case Active:
				task.status = Todo
			case Todo:
				// Already at min status, no change
			}
			return true
		}
		return false
	})
	m.autoSaveIfEnabled()
}

// createNewTaskBelow creates a new task below the currently selected task
func (m *Model) createNewTaskBelow() string {
	newTask := NewTask("", Todo)
	
	// Special case: if no tasks exist, add as first top-level task
	if len(m.tasks) == 0 || m.cursorID == "" {
		m.tasks = append(m.tasks, newTask)
		return newTask.id
	}
	
	parent, index := m.findParentTask(m.cursorID)
	if index < 0 {
		// If current task not found, add at end of top-level tasks
		m.tasks = append(m.tasks, newTask)
		return newTask.id
	}
	
	container := m.getTaskContainer(parent)
	
	// Insert after the current task
	insertTaskInSlice(container, index+1, newTask)
	
	// Auto-save will be called when the task title is set or when exiting edit mode
	return newTask.id
}

// createNewSubtask creates a new subtask at the end of the currently selected task's subtasks
func (m *Model) createNewSubtask() string {
	// Special case: if no tasks exist, create a top-level task instead
	if len(m.tasks) == 0 || m.cursorID == "" {
		return m.createNewTaskBelow()
	}
	
	currentTask := m.getCurrentTask()
	if currentTask == nil {
		// Fallback to creating a top-level task
		return m.createNewTaskBelow()
	}
	
	newTask := NewTask("", Todo)
	
	// Add to the end of the current task's subtasks
	currentTask.subtasks = append(currentTask.subtasks, newTask)
	
	// Auto-save will be called when the task title is set or when exiting edit mode
	return newTask.id
}

// deleteCurrentTask removes the currently selected task
func (m *Model) deleteCurrentTask() {
	parent, index := m.findParentTask(m.cursorID)
	if index < 0 {
		return // Task not found
	}
	
	container := m.getTaskContainer(parent)
	
	// Remove the task from its container
	removeTaskFromSlice(container, index)
	
	// Update cursor to a valid task
	m.updateCursorAfterDeletion()
	
	m.autoSaveIfEnabled()
}

// updateCursorAfterDeletion moves cursor to a valid task after deletion
func (m *Model) updateCursorAfterDeletion() {
	// First try to go back to the previously selected task
	if m.previousID != "" && m.findTaskByID(m.previousID) != nil {
		m.cursorID = m.previousID
		m.previousID = ""
		return
	}
	
	// Otherwise, select the first available task
	allIDs := m.getAllTaskIDs()
	if len(allIDs) > 0 {
		m.cursorID = allIDs[0]
	} else {
		m.cursorID = ""
	}
	m.previousID = ""
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
	
	m.autoSaveIfEnabled()
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
	
	m.autoSaveIfEnabled()
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
	
	m.autoSaveIfEnabled()
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
	
	m.autoSaveIfEnabled()
}

func (m Model) View() string {
	// Build title with task list indicator
	titleText := "Task Manager"
	if m.filePath != "" {
		taskListName := m.getTaskListDisplayName()
		titleText = fmt.Sprintf("Task Manager - %s", taskListName)
	}
	
	title := lipgloss.NewStyle().
		Render(titleText)

	// Compute inner content width to enable wrapping within the padded container
	innerWidth := m.width - TotalPadding*2
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

	// Add helpful message if no tasks exist
	if len(m.tasks) == 0 {
		helpText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")). // Gray
			Italic(true).
			Render("No tasks yet. Press 'n' to create your first task, or 'q' to quit.")
		rows = append(rows, "", helpText) // Empty line for spacing
	}

	// Add error message if present
	var bodyParts []string
	bodyParts = append(bodyParts, title)
	bodyParts = append(bodyParts, rows...)
	
	if m.showError {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")). // Red
			Background(lipgloss.Color("0")). // Black background
			Padding(0, 1).
			Margin(1, 0)
		
		errorMsg := errorStyle.Render("ERROR: " + m.lastError + " (Press ESC to dismiss)")
		bodyParts = append(bodyParts, errorMsg)
	}

	// Body: title + rows + error
	body := lipgloss.JoinVertical(lipgloss.Left, bodyParts...)

	// Wrap the entire body to inner width (mainly affects the title line).
	wrapped := lipgloss.NewStyle().
		Width(innerWidth).
		MaxWidth(innerWidth).
		Render(body)

	// Padded container with fixed outer width
	container := lipgloss.NewStyle().
		Padding(1, PaddingLeft).
		Width(innerWidth + TotalPadding*2).
		MaxWidth(innerWidth + TotalPadding*2).
		Render(wrapped)

	return container
}

func (m Model) renderRow(task Task, width int, indentLevel int, isSelected bool, isEditing bool) string {
	indent := m.renderIndentation(indentLevel)
	bulletRendered := m.renderBullet(task.status, isEditing, isSelected)
	cursorRendered := m.renderCursor(isSelected, isEditing)
	textColWidth := m.calculateTextWidth(width, indentLevel)
	textRendered := m.renderText(task, textColWidth, isSelected, isEditing)
	
	return lipgloss.JoinHorizontal(lipgloss.Top, cursorRendered, lipgloss.NewStyle().Render(indent), bulletRendered, textRendered)
}

func (m Model) renderIndentation(indentLevel int) string {
	indent := ""
	for i := 0; i < indentLevel-1; i++ {
		indent += strings.Repeat(" ", IndentWidth)
	}
	if indentLevel > 0 {
		indent += "╰ "
	}
	return indent
}

func (m Model) renderBullet(status TaskStatus, isEditing bool, isSelected bool) string {
	bulletMap := map[TaskStatus]string{
		Done:   "◉",
		Active: "◎",
		Todo:   "○",
	}
	style := lipgloss.NewStyle().Width(BulletWidth)
	if isEditing && !isSelected {
		style = style.Foreground(lipgloss.Color("8"))
	}
	return style.Render(bulletMap[status] + " ")
}

func (m Model) renderCursor(isSelected bool, isEditing bool) string {
	cursorSymbol := " "
	style := lipgloss.NewStyle().Width(CursorWidth)
	
	if isSelected {
		cursorSymbol = "▐"
		style = style.Foreground(lipgloss.Color("1")) // Red color
	}
	
	if isEditing && !isSelected {
		style = style.Foreground(lipgloss.Color("8"))
	}
	
	return style.Render(cursorSymbol + " ")
}

func (m Model) calculateTextWidth(width int, indentLevel int) int {
	textColWidth := width - CursorWidth - BulletWidth - (indentLevel * IndentWidth)
	if textColWidth < 0 {
		textColWidth = 0
	}
	return textColWidth
}

func (m Model) renderText(task Task, width int, isSelected bool, isEditing bool) string {
	if isEditing && isSelected {
		return lipgloss.NewStyle().Width(width).Render(m.textInput.View())
	}
	
	styleMap := map[TaskStatus]lipgloss.Style{
		Done:   lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Strikethrough(true),
		Active: lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		Todo:   lipgloss.NewStyle(),
	}
	
	style := styleMap[task.status]
	if isEditing && !isSelected {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	} else if isSelected && !isEditing {
		style = style.Underline(true)
	}
	
	text := style.Render(task.title)
	return lipgloss.NewStyle().Width(width).Render(text)
}

// loadTasksFromFile loads tasks from a file using the storage package
func loadTasksFromFile(filePath string) ([]Task, error) {
	taskData, err := storage.LoadTasks(filePath)
	if err != nil {
		return nil, err
	}
	
	return FromTaskDataSlice(taskData), nil
}

// saveTasksToFile saves tasks to a file using the storage package
func (m *Model) saveTasksToFile() error {
	if m.filePath == "" {
		return nil // No file path specified, skip saving
	}
	
	taskData := ToTaskDataSlice(m.tasks)
	return storage.SaveTasks(m.filePath, taskData)
}

// autoSaveIfEnabled saves tasks if auto-save is enabled
func (m *Model) autoSaveIfEnabled() {
	if m.autoSave {
		if err := m.saveTasksToFile(); err != nil {
			m.setError("Save failed: " + err.Error())
		} else {
			// Clear any previous error on successful save
			m.clearError()
		}
	}
}

// setError sets an error message to display to the user
func (m *Model) setError(message string) {
	m.lastError = message
	m.showError = true
}

// clearError clears any displayed error message
func (m *Model) clearError() {
	m.lastError = ""
	m.showError = false
}

// getTaskListDisplayName returns a user-friendly name for the current task list
func (m Model) getTaskListDisplayName() string {
	if m.filePath == "" {
		return "Untitled"
	}
	
	// Get the base filename without extension
	filename := filepath.Base(m.filePath)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	
	// Check if it's a global task list (in config directory)
	configDir := ""
	if homeDir, err := storage.GetConfigDir(); err == nil {
		configDir = filepath.Join(homeDir, "dotdot", "tasks")
	}
	
	if configDir != "" && strings.HasPrefix(m.filePath, configDir) {
		return fmt.Sprintf("%s (global)", name)
	}
	
	// For local files, show relative path if not in current directory
	if abs, err := filepath.Abs("."); err == nil {
		if strings.HasPrefix(m.filePath, abs) {
			// It's in current directory or subdirectory
			if rel, err := filepath.Rel(abs, m.filePath); err == nil && !strings.Contains(rel, "..") {
				if filepath.Dir(rel) == "." {
					return fmt.Sprintf("%s (local)", name)
				}
				return fmt.Sprintf("%s (local)", rel)
			}
		}
	}
	
	// For absolute paths or paths outside current directory
	return fmt.Sprintf("%s (%s)", name, filepath.Dir(m.filePath))
}

// Conversion functions between TUI Task and storage TaskData

// ToTaskData converts a TUI Task to a storage TaskData
func ToTaskData(task Task) storage.TaskData {
	subtasks := make([]storage.TaskData, len(task.Subtasks()))
	for i, subtask := range task.Subtasks() {
		subtasks[i] = ToTaskData(subtask)
	}

	return storage.TaskData{
		ID:       task.ID(),
		Title:    task.Title(),
		Status:   int(task.Status()),
		Subtasks: subtasks,
	}
}

// ToTaskDataSlice converts a slice of TUI Tasks to storage TaskData
func ToTaskDataSlice(tasks []Task) []storage.TaskData {
	taskData := make([]storage.TaskData, len(tasks))
	for i, task := range tasks {
		taskData[i] = ToTaskData(task)
	}
	return taskData
}

// FromTaskData converts a storage TaskData to a TUI Task
func FromTaskData(data storage.TaskData) Task {
	subtasks := make([]Task, len(data.Subtasks))
	for i, subtaskData := range data.Subtasks {
		subtasks[i] = FromTaskData(subtaskData)
	}

	return NewTaskWithID(data.ID, data.Title, TaskStatus(data.Status), subtasks...)
}

// FromTaskDataSlice converts a slice of storage TaskData to TUI Tasks
func FromTaskDataSlice(taskData []storage.TaskData) []Task {
	tasks := make([]Task, len(taskData))
	for i, data := range taskData {
		tasks[i] = FromTaskData(data)
	}
	return tasks
}

// Ensure Model implements tea.Model
var _ tea.Model = (*Model)(nil)
