package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"dotdot/internal/storage"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
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
	help           help.Model      // Help component
	keyMap         KeyMap          // Key bindings
	showFullHelp   bool            // Toggle between short and full help
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
	ti.Placeholder = "Task text..."
	ti.Prompt = ""

	var s textinput.Styles
	s.Cursor = textinput.CursorStyle{
		Shape: tea.CursorBar,
	}
	ti.SetStyles(s)
	ti.Focus()
	// ti.Cursor.Style = tea.CursorBar
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

	// Initialize help with custom styles
	helpModel := help.New()
	helpModel.Styles = GetHelpStyles()
	helpModel.Width = 80 // Default width, will be updated on first WindowSizeMsg

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
		help:           helpModel,
		keyMap:         DefaultKeyMap(),
		showFullHelp:   false,
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

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	return m, cmd
}

func (m Model) handleEditingMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keyMap.NewTaskBelowFromEdit):
		// Enter key: save current edit, then create new task below and enter edit mode
		// Special case: if current task is empty, delete it and enter normal mode
		if m.textInput.Value() == "" {
			m.deleteCurrentTask()
			m.editing = false
			m.textInput.Blur()
			return m, cmd
		}
		m.editTaskTitle(m.cursorID, m.textInput.Value())
		m.previousID = m.cursorID
		newTaskID := m.createNewTaskBelow()
		if newTaskID != "" {
			m.cursorID = newTaskID
			m.textInput.SetValue("")
			m.textInput.Focus()
		}
		return m, cmd
	case key.Matches(msg, m.keyMap.NewSubtaskFromEdit):
		// Shift+Enter: save current edit, then create new subtask and enter edit mode
		m.editTaskTitle(m.cursorID, m.textInput.Value())
		m.previousID = m.cursorID
		newTaskID := m.createNewSubtask()
		if newTaskID != "" {
			m.cursorID = newTaskID
			m.textInput.SetValue("")
			m.textInput.Focus()
		}
		return m, cmd
	case key.Matches(msg, m.keyMap.NewTaskInParentFromEdit):
		// Ctrl+Enter: save current edit, then create new task in parent and enter edit mode
		m.editTaskTitle(m.cursorID, m.textInput.Value())
		m.previousID = m.cursorID
		newTaskID := m.createNewTaskInParent()
		if newTaskID != "" {
			m.cursorID = newTaskID
			m.textInput.SetValue("")
			m.textInput.Focus()
		}
		return m, cmd
	case key.Matches(msg, m.keyMap.Cancel):
		// ESC: If the task title is empty, delete the task
		currentTask := m.getCurrentTask()
		if currentTask != nil && currentTask.title == "" {
			m.deleteCurrentTask()
		}
		m.editing = false
		m.textInput.Blur()
		return m, cmd
	}

	m.statusMessage = msg.String() // No blinking messages?
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keyMap.Cancel):
		// Clear error messages on ESC
		if m.showError {
			m.clearError()
			return m, nil
		}
		// If no error to clear, do nothing
	case key.Matches(msg, m.keyMap.Up):
		m.cursorID = m.getPreviousTaskID()
	case key.Matches(msg, m.keyMap.Down):
		m.cursorID = m.getNextTaskID()
	case key.Matches(msg, m.keyMap.Left):
		m.changeTaskStatusBackward()
	case key.Matches(msg, m.keyMap.Right):
		m.changeTaskStatusForward()
	case key.Matches(msg, m.keyMap.MoveUp):
		m.moveTaskUp()
	case key.Matches(msg, m.keyMap.MoveDown):
		m.moveTaskDown()
	case key.Matches(msg, m.keyMap.UnindentTask):
		m.unindentTask()
	case key.Matches(msg, m.keyMap.IndentTask):
		m.indentTask()
	case key.Matches(msg, m.keyMap.NewTaskBelow):
		m.previousID = m.cursorID
		newTaskID := m.createNewTaskBelow()
		if newTaskID != "" {
			m.cursorID = newTaskID
			m.editing = true
			m.textInput.SetValue("")
			m.textInput.Focus()
		}
		return m, nil
	case key.Matches(msg, m.keyMap.NewSubtask):
		m.previousID = m.cursorID
		newTaskID := m.createNewSubtask()
		if newTaskID != "" {
			m.cursorID = newTaskID
			m.editing = true
			m.textInput.SetValue("")
			m.textInput.Focus()
		}
		return m, nil
	case key.Matches(msg, m.keyMap.NewTaskInParent):
		m.previousID = m.cursorID
		newTaskID := m.createNewTaskInParent()
		if newTaskID != "" {
			m.cursorID = newTaskID
			m.editing = true
			m.textInput.SetValue("")
			m.textInput.Focus()
		}
		return m, nil
	case key.Matches(msg, m.keyMap.Undo):
		m.undo()
		return m, nil
	case key.Matches(msg, m.keyMap.Redo):
		m.redo()
		return m, nil
	case key.Matches(msg, m.keyMap.Copy):
		m.copyCurrentTaskToClipboard()
		return m, nil
	case key.Matches(msg, m.keyMap.Paste):
		m.pasteTaskFromClipboard()
		return m, nil
	case key.Matches(msg, m.keyMap.PasteAsSubtask):
		m.pasteTaskAsSubtask()
		return m, nil
	case key.Matches(msg, m.keyMap.Help):
		m.showFullHelp = !m.showFullHelp
		return m, nil
	case key.Matches(msg, m.keyMap.EditTask):
		m.editing = true
		task := m.getCurrentTask()
		if task != nil {
			m.textInput.SetValue(task.title)
		}
		m.textInput.Focus()
		return m, nil
	case key.Matches(msg, m.keyMap.DeleteTask):
		m.deleteCurrentTask()
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

	// Update help model width and build footer (error messages, status, and help)
	m.help.Width = innerWidth
	footerParts := m.buildFooterParts(innerWidth)

	var footer string
	if len(footerParts) > 0 {
		footer = lipgloss.NewStyle().
			Width(innerWidth).
			Render(lipgloss.JoinVertical(lipgloss.Left, footerParts...))
	}

	// Calculate viewport dimensions based on actual header and footer
	headerHeight := lipgloss.Height(header)
	footerHeight := 0
	if footer != "" {
		footerHeight = lipgloss.Height(footer)
	}

	viewportWidth := innerWidth
	viewportHeight := m.height - headerHeight - footerHeight - 2 // -2 for padding
	if viewportWidth < 0 {
		viewportWidth = 0
	}
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	// Update viewport dimensions
	m.viewport.SetWidth(viewportWidth)
	m.viewport.SetHeight(viewportHeight)

	// Build scrollable content (tasks)
	var rows []string
	cursorTaskPosition := 0
	cursorTaskFound := false

	// Get parent chain for underlining parent tasks
	parentChainIDs := m.getParentChainIDs(m.cursorID)

	// Helper function to recursively render tasks and subtasks
	var renderTasks func(tasks []Task, indentLevel int)
	renderTasks = func(tasks []Task, indentLevel int) {
		for _, task := range tasks {
			isSelected := task.id == m.cursorID
			row := m.renderRow(task, innerWidth, indentLevel, isSelected, m.editing, parentChainIDs)
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
	m.viewport.SetYOffset(viewportOffset)

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
		Padding(1, 0, 0, PaddingLeft).
		Width(m.width).
		MaxWidth(m.width).
		Render(view)

	return container
}

func (m Model) renderRow(task Task, width int, indentLevel int, isSelected bool, isEditing bool, parentChainIDs []string) string {
	indent := m.renderIndentation(indentLevel)
	bulletRendered := m.renderBullet(task.status, isEditing, isSelected)
	cursorRendered := m.renderCursor(isSelected, isEditing)
	textColWidth := m.calculateTextWidth(width, indentLevel)
	textRendered := m.renderText(task, textColWidth, isSelected, isEditing, parentChainIDs)

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

func (m Model) renderText(task Task, width int, isSelected bool, isEditing bool, parentChainIDs []string) string {
	if isEditing && isSelected {
		return lipgloss.NewStyle().Width(width).Render(m.textInput.View())
	}

	// Check if this task is a parent of the selected task
	isParentOfSelected := false
	for _, parentID := range parentChainIDs {
		if parentID == task.id {
			isParentOfSelected = true
			break
		}
	}

	style := GetTaskStyle(task.status)
	if isEditing && !isSelected {
		style = lipgloss.NewStyle().Foreground(lipgloss.Color(DimmedColor))
	} else if (isSelected || isParentOfSelected) && !isEditing {
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

// setStatus sets a status message
func (m *Model) setStatus(message string) {
	m.statusMessage = message
}

// clearStatus clears the status message
func (m *Model) clearStatus() {
	m.statusMessage = ""
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

// buildFooterParts builds all footer components (errors, status, help)
func (m Model) buildFooterParts(width int) []string {
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

	// Add help section
	var helpView string
	if m.showFullHelp {
		helpView = m.help.FullHelpView(m.keyMap.FullHelp())
	} else {
		helpView = m.help.ShortHelpView(m.keyMap.ShortHelp())
	}
	if helpView != "" {
		footerParts = append(footerParts, helpView)
	}

	return footerParts
}

// Ensure Model implements tea.Model
var _ tea.Model = (*Model)(nil)
