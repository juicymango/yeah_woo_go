package util

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Grep searches for files in dir containing the target string and returns their paths
func Grep(dir string, target string) ([]string, error) {
	var filesWithTarget []string // Store matching file paths

	// Walk the directory tree
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // Propagate any error encountered
		}
		if d.IsDir() {
			return nil // Skip directories
		}

		// Check if the current file contains the target string
		contains, err := FileContainsString(path, target)
		if err != nil {
			return err // Propagate any error encountered
		}
		if contains {
			filesWithTarget = append(filesWithTarget, path)
		}
		return nil
	})

	if err != nil {
		return nil, err // Return any errors encountered during the walk
	}

	return filesWithTarget, nil
}

// FileContainsString checks if a file contains a given string
func FileContainsString(filename, searchString string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), searchString) {
			return true, nil
		}
	}
	return false, scanner.Err()
}
