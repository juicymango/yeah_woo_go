package util

import (
	"go/ast"
	"go/printer"
	"go/token"
	"maps"
	"os"
	"slices"

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

func CloneNodeInfo(nodeInfo *model.NodeInfo) *model.NodeInfo {
	if nodeInfo == nil {
		return nil
	}
	newNodeListFields := make(map[string][]*model.NodeInfo, len(nodeInfo.NodeListFields))
	for name, nodes := range nodeInfo.NodeListFields {
		newNodeListFields[name] = slices.Clone(nodes)
	}
	newNodeInfo := &model.NodeInfo{
		Node:           nodeInfo.Node,
		Type:           nodeInfo.Type,
		NodeFields:     maps.Clone(nodeInfo.NodeFields),
		NodeListFields: newNodeListFields,
		StringFields:   maps.Clone(nodeInfo.StringFields),
		TokenFields:    maps.Clone(nodeInfo.TokenFields),
	}
	return newNodeInfo
}
