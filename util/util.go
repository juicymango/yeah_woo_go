package util

import (
	"encoding/json"
	"go/ast"
	"go/printer"
	"go/token"
	"log"
	"os"
	"slices"
	"sort"
	"strings"
	"unsafe"

	"github.com/juicymango/yeah_woo_go/model"
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
