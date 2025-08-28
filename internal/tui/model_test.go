package tui

import (
	"testing"
)

func TestTaskManipulation(t *testing.T) {
	// Create a model with minimal mock tasks for testing
	model := NewModel()
	model.tasks = GetMinimalMockTasks()
	
	// Set cursor to first task
	model.cursorID = model.tasks[0].id
	
	// Test moving task down
	originalFirstTaskID := model.tasks[0].id
	originalSecondTaskID := model.tasks[1].id
	model.moveTaskDown()
	
	// Verify the first task moved down
	if model.tasks[0].id != originalSecondTaskID {
		t.Errorf("Expected first task to be original second task, got different task")
	}
	if model.tasks[1].id != originalFirstTaskID {
		t.Errorf("Expected second task to be original first task, got different task")
	}

	// Test moving task up
	model.cursorID = model.tasks[1].id // Second position (which is now the original first task)
	model.moveTaskUp()
	
	// Verify the task moved back up
	if model.tasks[0].id != originalFirstTaskID {
		t.Errorf("Expected first task to be back to original first task")
	}
	if model.tasks[1].id != originalSecondTaskID {
		t.Errorf("Expected second task to be back to original second task")
	}

	// Test indenting a task (move it into the previous task)
	model.cursorID = model.tasks[1].id // Second task
	originalSecondTask := model.findTaskByID(model.cursorID)
	if originalSecondTask == nil {
		t.Fatal("Could not find second task")
	}
	originalSecondTaskTitle := originalSecondTask.title
	model.indentTask()
	
	// Verify the task was moved into the first task as a subtask
	if len(model.tasks[0].subtasks) == 0 {
		t.Error("Expected first task to have subtasks after indenting")
	}
	lastSubtask := model.tasks[0].subtasks[len(model.tasks[0].subtasks)-1]
	if lastSubtask.title != originalSecondTaskTitle {
		t.Errorf("Expected last subtask to be '%s', got '%s'", originalSecondTaskTitle, lastSubtask.title)
	}

	// Verify cursor is still on the same task (now a subtask)
	currentTask := model.getCurrentTask()
	if currentTask == nil || currentTask.title != originalSecondTaskTitle {
		t.Errorf("Expected cursor to follow indented task, got nil or different task")
	}

	// Test unindenting a task (move it back out)
	model.unindentTask()
	
	// Verify the task was moved back out
	if len(model.tasks) < 2 {
		t.Error("Expected at least 2 top-level tasks after unindenting")
	}
	// Find the task that was unindented (should be after the first task)
	found := false
	for i := 1; i < len(model.tasks); i++ {
		if model.tasks[i].title == originalSecondTaskTitle {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find task '%s' back at top level after unindenting", originalSecondTaskTitle)
	}
}

func TestTaskFinding(t *testing.T) {
	model := NewModel()
	model.tasks = GetMinimalMockTasks()

	// Test finding first task by ID
	firstTask := model.findTaskByID(model.tasks[0].id)
	if firstTask == nil {
		t.Fatal("Expected to find first task by ID")
	}
	if firstTask.title != "First task" {
		t.Errorf("Expected first task title to be 'First task', got '%s'", firstTask.title)
	}

	// Test finding a subtask by ID
	fourthTask := &model.tasks[3] // "Fourth task with subtasks"
	if len(fourthTask.subtasks) == 0 {
		t.Fatal("Expected fourth task to have subtasks")
	}
	firstSubtask := model.findTaskByID(fourthTask.subtasks[0].id)
	if firstSubtask == nil {
		t.Fatal("Expected to find first subtask by ID")
	}
	if firstSubtask.title != "Subtask 1" {
		t.Errorf("Expected subtask title to be 'Subtask 1', got '%s'", firstSubtask.title)
	}
}

func TestBoundaryConditions(t *testing.T) {
	model := NewModel()
	model.tasks = GetMinimalMockTasks()

	// Test moving first task up (should do nothing)
	model.cursorID = model.tasks[0].id
	originalTask := model.tasks[0].title
	model.moveTaskUp()
	if model.tasks[0].title != originalTask {
		t.Error("First task should not move when trying to move up")
	}

	// Test moving last task down (should do nothing)
	allIDs := model.getAllTaskIDs()
	lastTaskID := allIDs[len(allIDs)-1]
	model.cursorID = lastTaskID
	lastTask := model.getCurrentTask()
	if lastTask == nil {
		t.Fatal("Could not find last task")
	}
	lastTaskTitle := lastTask.title
	model.moveTaskDown()
	lastTaskAfter := model.getCurrentTask()
	if lastTaskAfter == nil || lastTaskAfter.title != lastTaskTitle {
		t.Error("Last task should not move when trying to move down")
	}

	// Test indenting first task (should do nothing as there's no previous task)
	model.cursorID = model.tasks[0].id
	originalSubtasks := len(model.tasks)
	model.indentTask()
	if len(model.tasks) != originalSubtasks {
		t.Error("First task should not be indentable")
	}

	// Test unindenting top-level task (should do nothing)
	model.cursorID = model.tasks[0].id
	originalTask = model.tasks[0].title
	model.unindentTask()
	if model.tasks[0].title != originalTask {
		t.Error("Top-level task should not be unindentable")
	}
}

func TestCursorPositioningDuringIndentation(t *testing.T) {
	model := NewModel()
	model.tasks = GetMinimalMockTasks()
	
	// Position cursor on the second task (index 1)
	if len(model.tasks) < 2 {
		t.Fatal("Need at least 2 tasks for this test")
	}
	model.cursorID = model.tasks[1].id
	secondTaskID := model.tasks[1].id
	secondTaskTitle := model.tasks[1].title
	
	// Indent the task
	model.indentTask()
	
	// Verify the cursor is still pointing to the same task ID (now a subtask)
	currentTask := model.getCurrentTask()
	if currentTask == nil {
		t.Fatal("Could not find current task after indentation")
	}
	if currentTask.id != secondTaskID {
		t.Errorf("Expected cursor to follow indented task by ID. Expected ID %s, got %s", secondTaskID, currentTask.id)
	}
	
	// Verify the task is now a subtask of the first task
	if len(model.tasks[0].subtasks) == 0 {
		t.Error("Expected first task to have subtasks after indentation")
	}
	
	// The indented task should be the last subtask
	lastSubtask := model.tasks[0].subtasks[len(model.tasks[0].subtasks)-1]
	if lastSubtask.title != secondTaskTitle {
		t.Errorf("Expected last subtask to be '%s', got '%s'", secondTaskTitle, lastSubtask.title)
	}
	
	// Verify the cursor ID matches the indented task ID
	if currentTask.id != lastSubtask.id {
		t.Error("Expected cursor to point to the same task instance after indentation")
	}
}