# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Library Requirements
- Use the V2 versions of the BubbleTea and related libraries from Charm

## Development Commands

### Building and Running
- `go run cmd/dotdot/main.go` - Run the TUI application
- `go build -o dotdot cmd/dotdot/main.go` - Build the binary

### Testing
- `go test ./internal/tui/` - Run all tests in the TUI package
- `go test -v ./internal/tui/` - Run tests with verbose output

## Architecture Overview

This is a terminal-based task management application built with Charm's BubbleTea framework. The application uses a hierarchical task structure with vim-style keybindings and supports persistent storage with global/local task lists.

### Project Structure

**Main Application (`cmd/dotdot/main.go`)**
- Entry point that handles CLI argument parsing and routing
- Routes commands to appropriate handlers: runTUI, listTasks, deleteTasks
- Uses the BubbleTea V2 program model for the TUI interface

**TUI Package (`internal/tui/`)**
- `model.go` - Core BubbleTea Model interface with `Update()`, `View()`, and `Init()`
- `operations.go` - Task CRUD operations, tree traversal, and task manipulation functions
- `styles.go` - All styling constants, color definitions, and pre-configured lipgloss styles
- `mock_tasks.go` - Sample data for testing and development

**Storage Package (`internal/storage/`)**
- `json.go` - File I/O operations, JSON serialization, and task list management
- Handles .dot file format with metadata (version, timestamps, task data)
- Supports backup creation and legacy format migration

**CLI Package (`internal/cli/`)**
- `args.go` - Command-line argument parsing and validation
- Supports global, local, and explicit file path operations

### Core Data Structures

**Model**: Main application state
- Task hierarchy, cursor position, editing state
- File path and auto-save configuration  
- Error handling state (lastError, showError)

**Task**: Hierarchical structure with UUID-based identification
- ID (string), title (string), status (TaskStatus enum), subtasks ([]Task)
- Status progression: Todo → Active → Done

**TaskStatus**: Enum with three states (Todo, Active, Done)

### State Management
- Tasks stored as hierarchical slice structure allowing unlimited nesting
- UUID-based task identification for cursor tracking and operations
- previousID maintained for smart cursor positioning after deletions
- Generic tree traversal utilities (`traverseTasks`, `modifyTaskByID`) for consistent operations

### Styling System
- Semantic color constants by use rather than color value:
  - `CursorColor`, `ActiveTaskColor`, `DimmedColor`, `ErrorTextColor`, etc.
- Pre-configured lipgloss styles for consistent UI elements
- Modular rendering functions: `renderRow`, `renderIndentation`, `renderBullet`, `renderCursor`, `renderText`

### Key Bindings Architecture
The application separates input handling into two modes:

**Normal Mode (`handleNormalMode`)**
- Navigation: k/j or up/down arrows
- Status changes: h/l or left/right arrows (Todo → Active → Done)
- Task operations: n (new task below), N (new subtask)  
- Task movement: ctrl+k/j or ctrl+up/down
- Task indentation: ctrl+h/l or ctrl+left/right
- Edit mode: Enter key

**Edit Mode (`handleEditingMode`)**
- Standard text input handling via bubbles/textinput
- Enter saves changes, ESC cancels
- ESC on empty tasks deletes them and returns cursor to previous selection

### Task Operations
- **Creation**: Unified `createTask(asSubtask bool)` handles both sibling and subtask creation
- **Movement**: Tasks can move up/down within their container, indent/unindent between hierarchy levels  
- **Status**: Unified `changeTaskStatus(direction int)` for three-state progression (Todo → Active → Done)
- **Deletion**: Empty tasks are auto-deleted on ESC during creation
- **Editing**: In-place text editing with visual feedback (underlining, dimming)

### File Storage System
- **Formats**: JSON-based .dot files with metadata (version, created/updated timestamps)
- **Global task lists**: Stored in `~/.config/dotdot/tasks/` 
- **Local task lists**: Stored in current working directory
- **Auto-save**: Automatic saving after every task operation when file path is configured
- **Backup system**: Creates .bak files before overwriting existing files
- **Legacy support**: Can load older format files and upgrade them on save

### Command Line Interface
- **Syntax**: `dotdot [flags] [command] [name]`
- **Commands**: `open` (default), `list`, `delete`
- **Global lists**: `dotdot open work` → `~/.config/dotdot/tasks/work.dot`
- **Local lists**: `dotdot --local open mytasks` → `./mytasks.dot`
- **Explicit paths**: `dotdot --file /path/to/tasks.dot open`

### Testing Structure
- Comprehensive test suite in `model_test.go` covering task manipulation, finding, boundary conditions, and cursor positioning
- Uses minimal mock data from `GetMinimalMockTasks()` for consistent test scenarios
- Tests verify both functional correctness and cursor behavior during operations