package logic

import (
	"go/ast"
	"strings"

	"github.com/juicymango/yeah_woo_go/model"
	"github.com/juicymango/yeah_woo_go/util"
)

func FilterRelevantNodeInfo(taskCtx *model.TaskCtx, nodeInfo *model.NodeInfo) (newNodeInfo *model.NodeInfo, isRelevant bool) {
	if nodeInfo == nil {
		return
	}

	newNodeInfo = util.CloneNodeInfo(nodeInfo)

	// Ident / SelectorExpr
	if nodeInfo.Type == "*ast.Ident" || nodeInfo.Type == "*ast.SelectorExpr" {
		expr := nodeInfo.Node.(ast.Expr)
		isRelevant = IsTargetVariable(taskCtx, expr)
		return
	}

	// Return
	if nodeInfo.Type == "*ast.ReturnStmt" && taskCtx.Input.ShowReturn {
		isRelevant = true
		return
	}

	// BlockStmt
	if nodeInfo.Type == "*ast.BlockStmt" {
		newNodeInfo.NodeListFields["List"] = newNodeInfo.NodeListFields["List"][:0]
		for _, fieldNodeInfo := range nodeInfo.NodeListFields["List"] {
			fieldNewNodeInfo, fieldIsRelevant := FilterRelevantNodeInfo(taskCtx, fieldNodeInfo)
			if fieldIsRelevant {
				isRelevant = true
				if fieldNewNodeInfo != nil {
					newNodeInfo.NodeListFields["List"] = append(newNodeInfo.NodeListFields["List"], fieldNewNodeInfo)
				}
			}
		}
		return
	}

	if nodeInfo.Type == "*ast.CaseClause" {
		newNodeInfo.NodeListFields["Body"] = newNodeInfo.NodeListFields["Body"][:0]
		for _, fieldNodeInfo := range nodeInfo.NodeListFields["Body"] {
			fieldNewNodeInfo, fieldIsRelevant := FilterRelevantNodeInfo(taskCtx, fieldNodeInfo)
			if fieldIsRelevant {
				isRelevant = true
				if fieldNewNodeInfo != nil {
					newNodeInfo.NodeListFields["Body"] = append(newNodeInfo.NodeListFields["Body"], fieldNewNodeInfo)
				}
			} else if fieldNodeInfo.Type == "*ast.ReturnStmt" {
				newNodeInfo.NodeListFields["Body"] = append(newNodeInfo.NodeListFields["Body"], fieldNewNodeInfo)
			}
		}
		return
	}

	// default
	for name, fieldNodeInfo := range nodeInfo.NodeFields {
		fieldNewNodeInfo, fieldIsRelevant := FilterRelevantNodeInfo(taskCtx, fieldNodeInfo)
		if fieldIsRelevant {
			isRelevant = true
		}
		if fieldNewNodeInfo != nil {
			newNodeInfo.NodeFields[name] = fieldNewNodeInfo
		}
	}
	for name, fieldNodeInfos := range nodeInfo.NodeListFields {
		for idx, fieldNodeInfo := range fieldNodeInfos {
			fieldNewNodeInfo, fieldIsRelevant := FilterRelevantNodeInfo(taskCtx, fieldNodeInfo)
			if fieldIsRelevant {
				isRelevant = true
			}
			if fieldNewNodeInfo != nil {
				newNodeInfo.NodeListFields[name][idx] = fieldNewNodeInfo
			}
		}
	}

	return
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
	for _, varName := range taskCtx.Input.VarNames {
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
