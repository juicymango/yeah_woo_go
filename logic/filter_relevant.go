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

	// BlockStmt
	// always return a non nil result
	if nodeInfo.Type == "*ast.BlockStmt" {
		newNodeInfo.NodeListFields["List"] = newNodeInfo.NodeListFields["List"][:0]
		for _, fieldNodeInfo := range nodeInfo.NodeListFields["List"] {
			fieldNewNodeInfo, fieldIsRelevant := FilterRelevantNodeInfo(taskCtx, fieldNodeInfo)
			if fieldIsRelevant {
				isRelevant = true
				if fieldNewNodeInfo != nil {
					newNodeInfo.NodeListFields["List"] = append(newNodeInfo.NodeListFields["List"], fieldNewNodeInfo)
				}
			} else if fieldNodeInfo.Type == "*ast.ReturnStmt" {
				newNodeInfo.NodeListFields["List"] = append(newNodeInfo.NodeListFields["List"], fieldNewNodeInfo)
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
	varName := taskCtx.Input.VarName
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name == varName
	case *ast.SelectorExpr:
		// If varName is in the form of "a.B.C", construct the full name from SelectorExpr
		fullVarName := GetSelectorExprFullName(x)
		return strings.HasPrefix(fullVarName, varName) || strings.HasPrefix(varName, fullVarName)
	default:
		return false
	}
}

// GetSelectorExprFullName recursively constructs the full variable name from a SelectorExpr,
// which can represent an expression like "a.B.C".
func GetSelectorExprFullName(expr *ast.SelectorExpr) string {
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
	return strings.Join(parts, ".")
}
