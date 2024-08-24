package util

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unsafe"

	"github.com/juicymango/yeah_woo_go/model"
)

var (
	versionRegexp *regexp.Regexp = regexp.MustCompile(`^v\d+(\.\d+)?`)
	nameRegexp    *regexp.Regexp = regexp.MustCompile(`[a-zA-Z0-9_]+$`)
)

// GetFunc searches for a function declaration with the given name in the specified file's AST.
// It returns the function declaration if found, and nil otherwise.
func GetFunc(f *ast.File, funcName string) *ast.FuncDecl {
	for _, decl := range f.Decls {
		// Check if the declaration is a function
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			// Check if the function has the correct name
			if funcDecl.Name.Name == funcName {
				return funcDecl
			}
		}
	}
	// Function not found; return nil
	return nil
}

func PrintFunc(fset *token.FileSet, funcDecl *ast.FuncDecl) {
	// Print the FuncDecl
	err := printer.Fprint(os.Stdout, fset, funcDecl)
	if err != nil {
		panic(err)
	}
}

func JsonString(v any) string {
	jsonBytes, jsonErr := json.Marshal(v)
	if jsonErr != nil {
		log.Printf("JsonString MarshalErr %+v", jsonErr)
		return ""
	}
	return unsafe.String(unsafe.SliceData(jsonBytes), len(jsonBytes))
}

// StringSliceToKey takes a slice of strings, sorts it, joins the sorted elements with a separator,
// and returns the result as a string. This string can be used as a map key.
func StringSliceToKey(slice []string) string {
	// Make a copy of the slice to avoid modifying the input slice
	sliceCopy := slices.Clone(slice)

	// Sort the copy of the slice
	sort.Strings(sliceCopy)

	// Join the sorted elements with a separator
	return strings.Join(sliceCopy, ",")
}

func GetFuncTaskKey(funcTask model.FuncTask) model.FuncTaskKey {
	return model.FuncTaskKey{
		Source:       funcTask.Source,
		FuncName:     funcTask.FuncName,
		VarNames:     StringSliceToKey(funcTask.VarNames),
		ShowReturn:   funcTask.ShowReturn,
		ShowBreak:    funcTask.ShowBreak,
		ShowContinue: funcTask.ShowContinue,
	}
}

func GetPackageNameFromPath(packagePath string) string {
	// Split the path into segments based on '/'
	segments := strings.Split(packagePath, "/")
	lastSegment := segments[len(segments)-1]

	// Handle versioning gracefully, ignoring it to focus on the actual package name
	if versionRegexp.MatchString(lastSegment) && len(segments) > 1 {
		// If the last segment is a version, consider the previous segment as the package name segment
		lastSegment = segments[len(segments)-2]
	}

	// Extract the longest suffix not containing special characters
	match := nameRegexp.FindStringSubmatch(lastSegment)
	if len(match) > 0 {
		return match[0]
	}

	// Default return if no match found (should not usually happen given the inputs)
	return ""
}

// RemoveQuotesIfPresent checks if the input string starts and ends with '"'
// and returns the string without these characters if present.
func RemoveQuotesIfPresent(input string) string {
	if len(input) >= 2 && strings.HasPrefix(input, `"`) && strings.HasSuffix(input, `"`) {
		return input[1 : len(input)-1]
	}
	return input
}

// GetAbsoluteImportPath returns the absolute path of a given import path.
func GetAbsoluteImportPath(importPath string) (string, error) {
	// Get GOPATH environment variable
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return "", fmt.Errorf("GOPATH environment variable is not set")
	}

	// Construct the absolute path
	absolutePath := filepath.Join(gopath, "src", importPath)
	return absolutePath, nil
}
