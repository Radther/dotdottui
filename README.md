# DotDotTUI

A terminal-based task management application built with Charm's BubbleTea framework. Features hierarchical task organization with vim-style keybindings and persistent storage with global/local task lists.

## Build Steps

1. **Prerequisites**: Ensure you have Go installed on your system
2. **Build the project**:
   ```bash
   go build -o dotdot cmd/dotdot/main.go
   ```
3. **Install to PATH**: Move the binary to a directory in your PATH:
   ```bash
   sudo mv dotdot /usr/local/bin/
   ```

## Usage

### Global Task Lists
```bash
dotdot open work              # Open global task list named "work"
dotdot list                   # List all global task lists
dotdot delete work            # Delete global task list named "work"
```

### Local Task Lists
```bash
dotdot --local open mytasks   # Open local task list named "mytasks"
dotdot --local list           # List all local task lists
dotdot --local delete mytasks # Delete local task list named "mytasks"
```

### Task List from File
```bash
dotdot --file /path/to/tasks.dot open  # Open task list from specific file path
```