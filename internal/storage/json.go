package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TaskData represents the serializable task structure
type TaskData struct {
	ID       string      `json:"id"`
	Title    string      `json:"title"`
	Status   int         `json:"status"`
	Subtasks []TaskData  `json:"subtasks"`
}

// FileData represents the complete file structure with metadata
type FileData struct {
	Version   string     `json:"version"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Tasks     []TaskData `json:"tasks"`
}

const CurrentVersion = "1.0.0"

// SaveTasks saves task data to a JSON file
func SaveTasks(filePath string, tasks []TaskData) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create backup of existing file
	if err := createBackup(filePath); err != nil {
		// Log error but don't fail the save operation
		fmt.Fprintf(os.Stderr, "Warning: failed to create backup: %v\n", err)
	}

	// Prepare file data
	fileData := FileData{
		Version:   CurrentVersion,
		CreatedAt: getCreationTime(filePath),
		UpdatedAt: time.Now(),
		Tasks:     tasks,
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks to JSON: %w", err)
	}

	// Write to temporary file first, then rename (atomic operation)
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file %s: %w", tempPath, err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		// Clean up temp file on failure
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temporary file to %s: %w", filePath, err)
	}

	return nil
}

// LoadTasks loads task data from a JSON file
func LoadTasks(filePath string) ([]TaskData, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Return empty task list for new files
		return []TaskData{}, nil
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Handle empty files
	if len(data) == 0 {
		return []TaskData{}, nil
	}

	// Try to parse as new format with metadata
	var fileData FileData
	if err := json.Unmarshal(data, &fileData); err != nil {
		// Fallback: try to parse as legacy format (just tasks array)
		var tasks []TaskData
		if legacyErr := json.Unmarshal(data, &tasks); legacyErr != nil {
			return nil, fmt.Errorf("failed to parse JSON file %s: %w (legacy parse also failed: %v)", filePath, err, legacyErr)
		}
		
		// Successfully parsed legacy format
		fmt.Fprintf(os.Stderr, "Warning: loaded legacy format file %s, will be upgraded on next save\n", filePath)
		return tasks, nil
	}

	// Validate version compatibility
	if fileData.Version != CurrentVersion {
		fmt.Fprintf(os.Stderr, "Warning: file %s has version %s, current version is %s\n", 
			filePath, fileData.Version, CurrentVersion)
	}

	return fileData.Tasks, nil
}

// ListGlobalTasks returns a list of available global task list names
func ListGlobalTasks() ([]string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	tasksDir := filepath.Join(configDir, "dotdot", "tasks")
	return listDotFiles(tasksDir)
}

// ListLocalTasks returns a list of available local task files in the current directory
func ListLocalTasks() ([]string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return listDotFiles(currentDir)
}

// DeleteTaskList deletes a task list file
func DeleteTaskList(filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("task list file %s does not exist", filePath)
	}

	// Create backup before deletion
	if err := createBackup(filePath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create backup before deletion: %v\n", err)
	}

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file %s: %w", filePath, err)
	}

	return nil
}

// Helper functions

func listDotFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // Directory doesn't exist, return empty list
		}
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var dotFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".dot") {
			// Remove .dot extension for display
			name := strings.TrimSuffix(entry.Name(), ".dot")
			dotFiles = append(dotFiles, name)
		}
	}

	return dotFiles, nil
}

func createBackup(filePath string) error {
	// Only create backup if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // No file to backup
	}

	backupPath := filePath + ".bak"
	
	// Read original file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Write backup
	return os.WriteFile(backupPath, data, 0644)
}

func getCreationTime(filePath string) time.Time {
	if stat, err := os.Stat(filePath); err == nil {
		return stat.ModTime() // Use ModTime as approximation for creation time
	}
	return time.Now()
}

func GetConfigDir() (string, error) {
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return configDir, nil
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	
	return filepath.Join(homeDir, ".config"), nil
}

// FileExists checks if a file exists
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// GetFileInfo returns basic information about a task file
func GetFileInfo(filePath string) (map[string]interface{}, error) {
	if !FileExists(filePath) {
		return nil, fmt.Errorf("file %s does not exist", filePath)
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	info := map[string]interface{}{
		"path":     filePath,
		"size":     stat.Size(),
		"modified": stat.ModTime(),
	}

	// Try to read version info
	data, err := os.ReadFile(filePath)
	if err == nil && len(data) > 0 {
		var fileData FileData
		if err := json.Unmarshal(data, &fileData); err == nil {
			info["version"] = fileData.Version
			info["created"] = fileData.CreatedAt
			info["task_count"] = len(fileData.Tasks)
		}
	}

	return info, nil
}