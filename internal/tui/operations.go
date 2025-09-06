package tui

import (
	"strings"

	"github.com/atotto/clipboard"
)

// Task manipulation and tree operations

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

// getParentChainIDs returns all parent task IDs from immediate parent up to root
func (m *Model) getParentChainIDs(taskID string) []string {
	var parentIDs []string
	currentTaskID := taskID
	
	for {
		parent, _ := m.findParentTask(currentTaskID)
		if parent == nil {
			break // Reached top level
		}
		parentIDs = append(parentIDs, parent.id)
		currentTaskID = parent.id
	}
	
	return parentIDs
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

// modifyCurrentTask applies a function to the currently selected task
func (m *Model) modifyCurrentTask(fn func(*Task)) {
	m.modifyTaskByID(m.cursorID, fn)
}

// modifyTaskByID applies a function to the task with the given ID
func (m *Model) modifyTaskByID(taskID string, fn func(*Task)) {
	m.traverseTasks(func(task *Task) bool {
		if task.id == taskID {
			fn(task)
			return true
		}
		return false
	})
	m.autoSaveIfEnabled()
}

func (m *Model) editTaskTitle(taskID string, newTitle string) {
	// Only take snapshot if title actually changed
	currentTask := m.findTaskByID(taskID)
	if currentTask != nil && currentTask.title != newTitle {
		m.takeSnapshot()
	}
	m.modifyTaskByID(taskID, func(task *Task) {
		task.title = newTitle
	})
}

// changeTaskStatus changes task status in the given direction
// direction: 1 for forward (Todo -> Active -> Done), -1 for backward (Done -> Active -> Todo)
func (m *Model) changeTaskStatus(direction int) {
	// Check if status will actually change
	currentTask := m.getCurrentTask()
	if currentTask == nil {
		return
	}

	willChange := false
	if direction > 0 {
		willChange = (currentTask.status == Todo) || (currentTask.status == Active)
	} else {
		willChange = (currentTask.status == Done) || (currentTask.status == Active)
	}

	if willChange {
		m.takeSnapshot()
	}

	m.modifyCurrentTask(func(task *Task) {
		if direction > 0 {
			// Forward: Todo -> Active -> Done
			switch task.status {
			case Todo:
				task.status = Active
			case Active:
				task.status = Done
			case Done:
				// Already at max status, no change
			}
		} else {
			// Backward: Done -> Active -> Todo
			switch task.status {
			case Done:
				task.status = Active
			case Active:
				task.status = Todo
			case Todo:
				// Already at min status, no change
			}
		}
	})
}

// changeTaskStatusForward advances task status: Todo -> Active -> Done
func (m *Model) changeTaskStatusForward() {
	m.changeTaskStatus(1)
}

// changeTaskStatusBackward reverses task status: Done -> Active -> Todo
func (m *Model) changeTaskStatusBackward() {
	m.changeTaskStatus(-1)
}

// createTask creates a new task at the specified location
// asSubtask: true to create as subtask, false to create as sibling
func (m *Model) createTask(asSubtask bool) string {
	// Take snapshot before creating task
	m.takeSnapshot()

	newTask := NewTask("", Todo)

	// Special case: if no tasks exist, add as first top-level task
	if len(m.tasks) == 0 || m.cursorID == "" {
		m.tasks = append(m.tasks, newTask)
		return newTask.id
	}

	if asSubtask {
		// Create as subtask
		currentTask := m.getCurrentTask()
		if currentTask == nil {
			// Fallback to creating a top-level task
			m.tasks = append(m.tasks, newTask)
			return newTask.id
		}

		// Add to the end of the current task's subtasks
		currentTask.subtasks = append(currentTask.subtasks, newTask)
		return newTask.id
	}

	// Create as sibling (below current task)
	parent, index := m.findParentTask(m.cursorID)
	if index < 0 {
		// If current task not found, add at end of top-level tasks
		m.tasks = append(m.tasks, newTask)
		return newTask.id
	}

	container := m.getTaskContainer(parent)

	// Insert after the current task
	insertTaskInSlice(container, index+1, newTask)

	return newTask.id
}

// createNewTaskBelow creates a new task below the currently selected task
func (m *Model) createNewTaskBelow() string {
	return m.createTask(false)
}

// createNewSubtask creates a new subtask at the end of the currently selected task's subtasks
func (m *Model) createNewSubtask() string {
	return m.createTask(true)
}

// createNewTaskInParent creates a new task in the parent of the currently selected task
func (m *Model) createNewTaskInParent() string {
	// Take snapshot before creating task
	m.takeSnapshot()

	newTask := NewTask("", Todo)

	// Special case: if no tasks exist, add as first top-level task
	if len(m.tasks) == 0 || m.cursorID == "" {
		m.tasks = append(m.tasks, newTask)
		return newTask.id
	}

	// Find the parent of the current task
	parent, _ := m.findParentTask(m.cursorID)

	if parent == nil {
		// Current task is at top level, create another top-level task at the end
		m.tasks = append(m.tasks, newTask)
		return newTask.id
	}

	// Current task has a parent, create a sibling at the parent's level
	grandparent, parentIndex := m.findParentTask(parent.id)
	if parentIndex < 0 {
		// Fallback to creating a top-level task
		m.tasks = append(m.tasks, newTask)
		return newTask.id
	}

	container := m.getTaskContainer(grandparent)

	// Insert after the parent task
	insertTaskInSlice(container, parentIndex+1, newTask)

	m.autoSaveIfEnabled()
	return newTask.id
}

// deleteCurrentTask removes the currently selected task
func (m *Model) deleteCurrentTask() {
	parent, index := m.findParentTask(m.cursorID)
	if index < 0 {
		return // Task not found
	}

	// Take snapshot before deletion
	m.takeSnapshot()

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

	// Take snapshot before moving
	m.takeSnapshot()

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

	// Take snapshot before moving
	m.takeSnapshot()

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

	// Take snapshot before unindenting
	m.takeSnapshot()

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

	// Take snapshot before indenting
	m.takeSnapshot()

	container := m.getTaskContainer(parent)
	// Get the previous sibling (which will become the parent)
	prevSibling := &(*container)[index-1]

	// Remove task from current location
	task := removeTaskFromSlice(container, index)

	// Add task as subtask of previous sibling
	prevSibling.subtasks = append(prevSibling.subtasks, task)

	m.autoSaveIfEnabled()
}

// takeSnapshot creates a snapshot of the current model state
func (m *Model) takeSnapshot() {
	// Create a deep copy of tasks
	tasksCopy := make([]Task, len(m.tasks))
	copy(tasksCopy, m.tasks)
	tasksCopy = m.deepCopyTasks(tasksCopy)

	snapshot := ModelSnapshot{
		tasks:      tasksCopy,
		cursorID:   m.cursorID,
		previousID: m.previousID,
	}

	// Add to undo stack
	m.undoStack = append(m.undoStack, snapshot)

	// Limit history size
	if len(m.undoStack) > m.maxHistorySize {
		m.undoStack = m.undoStack[1:]
	}

	// Clear redo stack when new operation is performed
	m.redoStack = m.redoStack[:0]
}

// deepCopyTasks creates a deep copy of a task slice
func (m *Model) deepCopyTasks(tasks []Task) []Task {
	result := make([]Task, len(tasks))
	for i, task := range tasks {
		result[i] = Task{
			id:       task.id,
			title:    task.title,
			status:   task.status,
			subtasks: m.deepCopyTasks(task.subtasks),
		}
	}
	return result
}

// undo restores the last state from undo stack
func (m *Model) undo() {
	if len(m.undoStack) == 0 {
		return
	}

	// Save current state to redo stack
	currentSnapshot := ModelSnapshot{
		tasks:      m.deepCopyTasks(m.tasks),
		cursorID:   m.cursorID,
		previousID: m.previousID,
	}
	m.redoStack = append(m.redoStack, currentSnapshot)

	// Limit redo stack size
	if len(m.redoStack) > m.maxHistorySize {
		m.redoStack = m.redoStack[1:]
	}

	// Restore from undo stack
	snapshot := m.undoStack[len(m.undoStack)-1]
	m.undoStack = m.undoStack[:len(m.undoStack)-1]

	m.tasks = snapshot.tasks
	m.cursorID = snapshot.cursorID
	m.previousID = snapshot.previousID

	m.autoSaveIfEnabled()
}

// redo restores the last state from redo stack
func (m *Model) redo() {
	if len(m.redoStack) == 0 {
		return
	}

	// Save current state to undo stack
	currentSnapshot := ModelSnapshot{
		tasks:      m.deepCopyTasks(m.tasks),
		cursorID:   m.cursorID,
		previousID: m.previousID,
	}
	m.undoStack = append(m.undoStack, currentSnapshot)

	// Limit undo stack size
	if len(m.undoStack) > m.maxHistorySize {
		m.undoStack = m.undoStack[1:]
	}

	// Restore from redo stack
	snapshot := m.redoStack[len(m.redoStack)-1]
	m.redoStack = m.redoStack[:len(m.redoStack)-1]

	m.tasks = snapshot.tasks
	m.cursorID = snapshot.cursorID
	m.previousID = snapshot.previousID

	m.autoSaveIfEnabled()
}

// copyCurrentTaskToClipboard copies the current task's title to the system clipboard
func (m *Model) copyCurrentTaskToClipboard() {
	task := m.getCurrentTask()
	if task == nil {
		m.setStatus("No task selected to copy")
		return
	}

	if err := clipboard.WriteAll(task.title); err != nil {
		m.setError("Failed to copy to clipboard: " + err.Error())
		return
	}

	m.setStatus("Task copied to clipboard")
	m.clearError()
}

// pasteTaskFromClipboard creates a new task below current position using clipboard contents
func (m *Model) pasteTaskFromClipboard() {
	clipContent, err := clipboard.ReadAll()
	if err != nil {
		m.setError("Failed to read from clipboard: " + err.Error())
		return
	}

	if strings.TrimSpace(clipContent) == "" {
		m.setStatus("Clipboard is empty")
		return
	}

	// Reuse existing task creation infrastructure
	m.previousID = m.cursorID
	newTaskID := m.createNewTaskBelow()
	if newTaskID != "" {
		// Set the task title to clipboard contents
		m.editTaskTitle(newTaskID, strings.TrimSpace(clipContent))
		m.cursorID = newTaskID
		m.setStatus("Task pasted from clipboard")
		m.clearError()
	}
}

// pasteTaskAsSubtask creates a new subtask using clipboard contents
func (m *Model) pasteTaskAsSubtask() {
	clipContent, err := clipboard.ReadAll()
	if err != nil {
		m.setError("Failed to read from clipboard: " + err.Error())
		return
	}

	if strings.TrimSpace(clipContent) == "" {
		m.setStatus("Clipboard is empty")
		return
	}

	// Reuse existing subtask creation infrastructure
	m.previousID = m.cursorID
	newTaskID := m.createNewSubtask()
	if newTaskID != "" {
		// Set the task title to clipboard contents
		m.editTaskTitle(newTaskID, strings.TrimSpace(clipContent))
		m.cursorID = newTaskID
		m.setStatus("Subtask pasted from clipboard")
		m.clearError()
	}
}
