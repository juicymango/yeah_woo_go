package logic

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"slices"
	"strings"

	"github.com/juicymango/yeah_woo_go/model"
	"github.com/juicymango/yeah_woo_go/util"
)

func FilterRelevantNodeInfo(taskCtx *model.TaskCtx, nodeInfo *model.NodeInfo) *model.NodeInfo {
	if nodeInfo == nil {
		return nil
	}

	// default
	newNodeInfo := util.CloneNodeInfo(nodeInfo)
	newNodeInfo.RelevantTaskResult = &model.RelevantTaskResult{}
	for name, fieldNodeInfo := range nodeInfo.NodeFields {
		fieldNewNodeInfo := FilterRelevantNodeInfo(taskCtx, fieldNodeInfo)
		if fieldNewNodeInfo == nil {
			continue
		}
		newNodeInfo.NodeFields[name] = fieldNewNodeInfo
		if fieldNewNodeInfo.RelevantTaskResult != nil {
			if fieldNewNodeInfo.RelevantTaskResult.IsRelevant {
				newNodeInfo.RelevantTaskResult.IsRelevant = true
			}
			if fieldNewNodeInfo.RelevantTaskResult.NotFilterByBlock {
				newNodeInfo.RelevantTaskResult.NotFilterByBlock = true
			}
		}
	}
	for name, fieldNodeInfos := range nodeInfo.NodeListFields {
		for idx, fieldNodeInfo := range fieldNodeInfos {
			fieldNewNodeInfo := FilterRelevantNodeInfo(taskCtx, fieldNodeInfo)
			if fieldNewNodeInfo == nil {
				continue
			}
			newNodeInfo.NodeListFields[name][idx] = fieldNewNodeInfo
			if fieldNewNodeInfo.RelevantTaskResult.IsRelevant {
				newNodeInfo.RelevantTaskResult.IsRelevant = true
			}
			if fieldNewNodeInfo.RelevantTaskResult.NotFilterByBlock {
				newNodeInfo.RelevantTaskResult.NotFilterByBlock = true
			}
		}
	}

	// Ident / SelectorExpr
	if nodeInfo.Type == "*ast.Ident" || nodeInfo.Type == "*ast.SelectorExpr" {
		expr := nodeInfo.Node.(ast.Expr)
		newNodeInfo.RelevantTaskResult.IsRelevant = IsTargetVariable(taskCtx, expr)
		return newNodeInfo
	}

	// Return
	if nodeInfo.Type == "*ast.ReturnStmt" && taskCtx.Input.FuncTask.ShowReturn {
		newNodeInfo.RelevantTaskResult.NotFilterByBlock = true
		return newNodeInfo
	}
	// Break
	if nodeInfo.Type == "*ast.BranchStmt" && nodeInfo.TokenFields["Tok"] == "break" && taskCtx.Input.FuncTask.ShowBreak {
		newNodeInfo.RelevantTaskResult.NotFilterByBlock = true
		return newNodeInfo
	}
	// Continue
	if nodeInfo.Type == "*ast.BranchStmt" && nodeInfo.TokenFields["Tok"] == "continue" && taskCtx.Input.FuncTask.ShowContinue {
		newNodeInfo.RelevantTaskResult.NotFilterByBlock = true
		return newNodeInfo
	}

	// BlockStmt / CaseClause
	if nodeInfo.Type == "*ast.BlockStmt" || nodeInfo.Type == "*ast.CaseClause" {
		fieldName := "List"
		if nodeInfo.Type == "*ast.CaseClause" {
			fieldName = "Body"
		}
		newNodeInfo.NodeListFields[fieldName] = slices.DeleteFunc(newNodeInfo.NodeListFields[fieldName], func(fieldNodeInfo *model.NodeInfo) bool {
			if fieldNodeInfo == nil {
				return true
			}
			if fieldNodeInfo.RelevantTaskResult == nil {
				return false
			}
			return !fieldNodeInfo.RelevantTaskResult.IsRelevant && !fieldNodeInfo.RelevantTaskResult.NotFilterByBlock
		})
		return newNodeInfo
	}

	// CallExpr
	if nodeInfo.Type == "*ast.CallExpr" {
		FilterRelevantCallExpr(taskCtx, newNodeInfo)
		return newNodeInfo
	}

	return newNodeInfo
}

func IsTargetVariable(taskCtx *model.TaskCtx, expr ast.Expr) bool {
	var nameParts []string
	switch x := expr.(type) {
	case *ast.Ident:
		nameParts = []string{x.Name}
	case *ast.SelectorExpr:
		// If varName is in the form of "a.B.C", construct the full name from SelectorExpr
		nameParts = GetSelectorExprNameParts(x)
	default:
		return false
	}
	for _, varName := range taskCtx.Input.FuncTask.VarNames {
		match := true
		varNameParts := strings.Split(varName, ".")
		for i := 0; i < len(varNameParts) && i < len(nameParts); i++ {
			if nameParts[i] != varNameParts[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// GetSelectorExprNameParts recursively constructs the full variable name from a SelectorExpr,
// which can represent an expression like "a.B.C".
func GetSelectorExprNameParts(expr *ast.SelectorExpr) []string {
	var parts []string
	for expr != nil {
		parts = append([]string{expr.Sel.Name}, parts...)
		x, ok := expr.X.(*ast.SelectorExpr)
		if !ok {
			if ident, ok := expr.X.(*ast.Ident); ok {
				parts = append([]string{ident.Name}, parts...)
			}
			break
		}
		expr = x
	}
	return parts
}

func GetFuncNodeInfo(taskCtx *model.TaskCtx) *model.NodeInfo {
	// Create a new token file set which is needed for parsing
	if taskCtx.FileSet == nil {
		taskCtx.FileSet = token.NewFileSet()
	}

	// Parse the file containing the Go program
	fileNode, err := parser.ParseFile(taskCtx.FileSet, taskCtx.Input.FuncTask.Source, nil, parser.ParseComments)
	if err != nil {
		log.Printf("GetFuncNodeInfo ParseFileErr, err:%+v, task:%+v", err, util.JsonString(&taskCtx.Input.FuncTask))
		return nil
	}

	funcDecl := util.GetFunc(fileNode, taskCtx.Input.FuncTask.FuncName)
	if funcDecl == nil {
		log.Printf("GetFuncNodeInfo GetFuncFail, task:%+v", util.JsonString(&taskCtx.Input.FuncTask))
		return nil
	}
	nodeInfo := util.GetNodeInfo(funcDecl)
	return nodeInfo
}
