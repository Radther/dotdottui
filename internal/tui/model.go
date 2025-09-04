package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"dotdot/internal/storage"

	"github.com/charmbracelet/bubbles/v2/textinput"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/google/uuid"
)

type Model struct {
	width          int
	height         int
	tasks          []Task
	cursorID       string
	previousID     string
	editing        bool
	textInput      textinput.Model
	viewport       viewport.Model
	filePath       string          // Path to the current task file
	autoSave       bool            // Enable auto-save after operations
	lastError      string          // Last error message to display
	showError      bool            // Whether to show the error message
	undoStack      []ModelSnapshot // History for undo operations
	redoStack      []ModelSnapshot // History for redo operations
	maxHistorySize int             // Maximum number of history entries
	statusMessage  string          // Debug/status message to display
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

// ModelSnapshot represents a state snapshot for undo/redo functionality
type ModelSnapshot struct {
	tasks      []Task
	cursorID   string
	previousID string
}

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

	// Initialize viewport
	vp := viewport.New(
		viewport.WithWidth(80),
		viewport.WithHeight(24),
	) // Default size, will be updated on first WindowSizeMsg

	return Model{
		tasks:          tasks,
		cursorID:       cursorID,
		previousID:     "",
		editing:        false,
		textInput:      ti,
		viewport:       vp,
		filePath:       filePath,
		autoSave:       filePath != "", // Enable auto-save when file path is provided
		lastError:      loadError,
		showError:      loadError != "",
		undoStack:      make([]ModelSnapshot, 0),
		redoStack:      make([]ModelSnapshot, 0),
		maxHistorySize: 50,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Reserve space for title (2 lines) and error message if shown (2 lines) and status (1 line)
		headerHeight := 2
		footerHeight := 0
		if m.showError {
			footerHeight += 2
		}
		if m.statusMessage != "" {
			footerHeight += 1
		}

		// Calculate viewport dimensions
		viewportWidth := m.width - TotalPadding
		viewportHeight := m.height - headerHeight - footerHeight - 2 // -2 for padding
		if viewportWidth < 0 {
			viewportWidth = 0
		}
		if viewportHeight < 0 {
			viewportHeight = 0
		}

		// Update viewport dimensions
		m.viewport = viewport.New(
			viewport.WithWidth(viewportWidth),
			viewport.WithHeight(viewportHeight),
		)

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
	case "shift+enter": // Currently unworking
		// Save current edit, then create new task below and enter edit mode
		m.statusMessage = "Shift+Enter pressed - creating task below"
		m.editTaskTitle(m.cursorID, m.textInput.Value())
		m.previousID = m.cursorID
		newTaskID := m.createNewTaskBelow()
		if newTaskID != "" {
			m.cursorID = newTaskID
			m.textInput.SetValue("")
			m.textInput.Focus()
		}
		return m, cmd
	case "ctrl+enter": // Currently unworking
		// Save current edit, then create new task in parent and enter edit mode
		m.statusMessage = "Ctrl+Enter pressed - creating task in parent"
		m.editTaskTitle(m.cursorID, m.textInput.Value())
		m.previousID = m.cursorID
		newTaskID := m.createNewTaskInParent()
		if newTaskID != "" {
			m.cursorID = newTaskID
			m.textInput.SetValue("")
			m.textInput.Focus()
		}
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
	case "ctrl+n":
		m.previousID = m.cursorID
		newTaskID := m.createNewTaskInParent()
		if newTaskID != "" {
			m.cursorID = newTaskID
			m.editing = true
			m.textInput.SetValue("")
			m.textInput.Focus()
		}
		return m, nil
	case "u":
		m.undo()
		return m, nil
	case "r":
		m.redo()
		return m, nil
	case "y":
		m.copyCurrentTaskToClipboard()
		return m, nil
	case "p":
		m.pasteTaskFromClipboard()
		return m, nil
	case "shift+p", "P":
		m.pasteTaskAsSubtask()
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

func (m Model) View() string {
	// Calculate inner width for content
	innerWidth := m.width - TotalPadding
	if innerWidth < 0 {
		innerWidth = 0
	}

	// Build header (title)
	titleText := "Task Manager"
	if m.filePath != "" {
		titleText = m.getTaskListDisplayName()
	}
	header := lipgloss.NewStyle().
		Width(innerWidth).
		Render(titleText)

	// Build scrollable content (tasks)
	var rows []string
	cursorTaskPosition := 0
	cursorTaskFound := false

	// Helper function to recursively render tasks and subtasks
	var renderTasks func(tasks []Task, indentLevel int)
	renderTasks = func(tasks []Task, indentLevel int) {
		for _, task := range tasks {
			isSelected := task.id == m.cursorID
			row := m.renderRow(task, innerWidth, indentLevel, isSelected, m.editing)
			if !cursorTaskFound {
				cursorTaskPosition += lipgloss.Height(row)
				if isSelected {
					cursorTaskFound = true
				}
			}
			rows = append(rows, row)
			if len(task.subtasks) > 0 {
				renderTasks(task.subtasks, indentLevel+1)
			}
		}
	}

	renderTasks(m.tasks, 0)

	// Add helpful message if no tasks exist
	if len(m.tasks) == 0 {
		helpText := HelpStyle.Render("No tasks yet. Press 'n' to create your first task, or 'q' to quit.")
		rows = append(rows, "", helpText) // Empty line for spacing
	}

	// Set viewport content
	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	m.viewport.SetContent(content)
	viewportOffset := 0
	if cursorTaskPosition > m.viewport.Height()-2 {
		viewportOffset = cursorTaskPosition - (m.viewport.Height() - 2)
	}
	m.viewport.YOffset = viewportOffset

	// Build footer (error messages and status)
	var footerParts []string
	if m.showError {
		errorMsg := ErrorStyle.Render("ERROR: " + m.lastError + " (Press ESC to dismiss)")
		footerParts = append(footerParts, errorMsg)
	}
	if m.statusMessage != "" {
		statusMsg := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("Status: " + m.statusMessage)
		footerParts = append(footerParts, statusMsg)
	}

	var footer string
	if len(footerParts) > 0 {
		footer = lipgloss.NewStyle().
			Width(innerWidth).
			Render(lipgloss.JoinVertical(lipgloss.Left, footerParts...))
	}

	// Combine header, viewport, and footer
	var viewParts []string
	viewParts = append(viewParts, header)
	viewParts = append(viewParts, m.viewport.View())
	if footer != "" {
		viewParts = append(viewParts, footer)
	}

	view := lipgloss.JoinVertical(lipgloss.Left, viewParts...)

	// Wrap in padded container
	container := lipgloss.NewStyle().
		Padding(1, PaddingLeft).
		Width(m.width).
		MaxWidth(m.width).
		Render(view)

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
	style := BulletStyle
	if isEditing && !isSelected {
		style = BulletDimmedStyle
	}
	return style.Render(BulletSymbols[status] + " ")
}

func (m Model) renderCursor(isSelected bool, isEditing bool) string {
	cursorSymbol := " "
	style := CursorStyle

	if isSelected {
		cursorSymbol = "▐"
		style = CursorSelectedStyle
	} else if isEditing && !isSelected {
		style = CursorDimmedStyle
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

	style := GetTaskStyle(task.status)
	if isEditing && !isSelected {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(DimmedColor))
	} else if isSelected && !isEditing {
		style = style.Underline(true)
	}

	// Apply width constraints and styling in one operation to ensure proper wrapping
	return style.Width(width).Render(task.title)
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
