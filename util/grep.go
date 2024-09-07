package util

import (
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"
)

func Grep1(dir string, target string) ([]string, error) {
	// Construct the grep command
	// FIXME: cannot use * here
	cmd := exec.Command("grep", "-l", "-d", "skip", target, filepath.Join(dir, "*.go"))

	// Execute the command
	output, err := cmd.Output()
	if err != nil {
		// if "grep" returns no matches, it exits with a non-zero status, we handle it separately
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return nil, nil // no matches found
		}
		log.Printf("GrepErr dir:%s, target:%s, err:%+v, p:%s", dir, target, err, filepath.Join(dir, "*.go"))
		return nil, err
	}

	log.Printf("GrepSucc dir:%s, target:%s, err:%+v, p:%s", dir, target, err, filepath.Join(dir, "*.go"))

	// Split the output into lines, each line should be a file path
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Convert to absolute paths
	var filePaths []string
	for _, line := range lines {
		if absPath, err := filepath.Abs(line); err == nil {
			filePaths = append(filePaths, absPath)
		} else {
			// Handle error if filepath.Abs fails
			return nil, err
		}
	}

	// Return the list of file paths
	return filePaths, nil
}

// Grep searches for files in dir containing the target string and returns their paths
func Grep(dir string, target string) ([]string, error) {
	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	var filesWithTarget []string // Store matching file paths

	// Walk the directory tree
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // Propagate any error encountered
		}
		if d.IsDir() {
			pathAbs, _ := filepath.Abs(path)
			if pathAbs != dirAbs {
				return filepath.SkipDir // Skip directories
			}
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
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
	log.Printf("Grep dir:%s, target:%s, err:%+v, filesWithTarget:%+v", dir, target, err, filesWithTarget)

	if err != nil {
		return nil, err // Return any errors encountered during the walk
	}

	return filesWithTarget, nil
}

// FileContainsString checks if a file contains a given string
func FileContainsString(filename, searchString string) (bool, error) {
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("FileContainsString ReadFileErr %+v", err)
		return false, err
	}
	fileStr := unsafe.String(unsafe.SliceData(fileBytes), len(fileBytes))
	if strings.Contains(fileStr, searchString) {
		return true, nil
	}
	return false, nil
}
