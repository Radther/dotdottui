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

This is a terminal-based task management application built with Charm's BubbleTea framework. The application uses a hierarchical task structure with vim-style keybindings.

### Core Components

**Main Application (`cmd/dotdot/main.go`)**
- Entry point that initializes the TUI with `tea.WithAltScreen()`
- Uses the BubbleTea V2 program model

**TUI Model (`internal/tui/model.go`)**
- Implements the core BubbleTea Model interface with `Update()`, `View()`, and `Init()`
- Manages application state including task hierarchy, cursor position, and editing mode
- Key data structures:
  - `Model`: Main application state with width, height, tasks, cursorID, previousID, editing flag, and textInput
  - `Task`: Hierarchical structure with id (UUID), title, status (Todo/Active/Done), and subtasks slice
  - UI constants: CursorWidth, BulletWidth, IndentWidth, PaddingLeft/Right, TotalPadding

**State Management**
- Tasks are stored as a hierarchical slice structure allowing unlimited nesting
- Uses UUID-based task identification for cursor tracking and operations
- Maintains previousID for smart cursor positioning after deletions
- Generic tree traversal utility (`traverseTasks`) used across multiple operations

**Rendering System**
- Modular rendering with separate functions: `renderRow`, `renderIndentation`, `renderBullet`, `renderCursor`, `renderText`
- Handles edit mode styling (dimming non-selected tasks, removing text input prompt)
- Uses lipgloss V2 for styling with color codes (1=red cursor, 2=green active, 8=gray dimmed)

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
- **Creation**: New tasks created with empty title, Todo status, auto-generated UUID
- **Movement**: Tasks can move up/down within their container, indent/unindent between hierarchy levels
- **Status**: Three-state progression (Todo → Active → Done) with left/right arrow control
- **Deletion**: Empty tasks are auto-deleted on ESC during creation
- **Editing**: In-place text editing with visual feedback (underlining, dimming)

### Testing Structure
- Comprehensive test suite in `model_test.go` covering task manipulation, finding, boundary conditions, and cursor positioning
- Uses minimal mock data from `GetMinimalMockTasks()` for consistent test scenarios
- Tests verify both functional correctness and cursor behavior during operations