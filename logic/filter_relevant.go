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
		if newNodeInfo.RelevantTaskResult.IsRelevant {
			return newNodeInfo
		}
		expr := nodeInfo.Node.(ast.Expr)
		newNodeInfo.RelevantTaskResult.IsRelevant = IsTargetVariable(taskCtx, expr)
		log.Printf("FilterRelevantNodeInfo Ident / SelectorExpr, node:%+v, IsRelevant:%+v", util.JsonString(nodeInfo), newNodeInfo.RelevantTaskResult.IsRelevant)
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
	if nodeInfo.Type == "*ast.CallExpr" && taskCtx.Input.FuncTask.EnableCall {
		FilterRelevantCallExpr(taskCtx, newNodeInfo)
		return newNodeInfo
	}

	// FuncDecl
	if nodeInfo.Type == "*ast.FuncDecl" {
		FilterRelevantFuncCalls(taskCtx, nodeInfo)
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
		varNameParts := strings.Split(varName, ".")
		if taskCtx.Input.FuncTask.ExactMatch && len(varNameParts) != len(nameParts) {
			continue
		}
		match := true
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
	fileInfo := GetFileInfo(taskCtx)
	if fileInfo == nil {
		log.Printf("GetFuncNodeInfo fileInfo nil, task:%+v", util.JsonString(&taskCtx.Input.FuncTask))
		return nil
	}
	funcNode := fileInfo.FuncMap[model.FuncKey{
		RecvTypes: taskCtx.Input.FuncTask.RecvTypes,
		Name:      taskCtx.Input.FuncTask.FuncName,
	}]
	if funcNode == nil {
		log.Printf("GetFuncNodeInfo funcNode nil, task:%+v", util.JsonString(&taskCtx.Input.FuncTask))
		return nil
	}
	return funcNode
}

func GetFileInfo(taskCtx *model.TaskCtx) *model.FileInfo {
	if taskCtx.FileInfoMap[taskCtx.Input.FuncTask.Source] != nil {
		return taskCtx.FileInfoMap[taskCtx.Input.FuncTask.Source]
	}

	if taskCtx.FileInfoMap == nil {
		taskCtx.FileInfoMap = make(map[string]*model.FileInfo)
	}
	// Create a new token file set which is needed for parsing
	if taskCtx.FileSet == nil {
		taskCtx.FileSet = token.NewFileSet()
	}

	// Parse the file containing the Go program
	fileNode, err := parser.ParseFile(taskCtx.FileSet, taskCtx.Input.FuncTask.Source, nil, parser.ParseComments)
	if err != nil {
		log.Printf("GetFileInfo ParseFileErr, err:%+v, task:%+v", err, util.JsonString(&taskCtx.Input.FuncTask))
		return nil
	}
	nodeInfo := util.GetNodeInfo(fileNode)
	fileInfo := &model.FileInfo{
		NodeInfo: nodeInfo,
		Package:  nodeInfo.NodeFields["Name"].StringFields["Name"],
	}
	taskCtx.FileInfoMap[taskCtx.Input.FuncTask.Source] = fileInfo

	GetFileInfoFuncMap(taskCtx, fileInfo)
	GetFileInfoImportMap(taskCtx, fileInfo)
	return fileInfo
}

func GetFileInfoFuncMap(taskCtx *model.TaskCtx, fileInfo *model.FileInfo) {
	decls, ok := fileInfo.NodeInfo.NodeListFields["Decls"]
	if !ok {
		return
	}
	fileInfo.FuncMap = make(map[model.FuncKey]*model.NodeInfo)
	for _, decl := range decls {
		if decl.Type != "*ast.FuncDecl" {
			continue
		}
		funcName := decl.NodeFields["Name"].StringFields["Name"]
		recvTypes := make([]string, 0)
		if decl.NodeFields["Recv"] != nil {
			for _, recv := range decl.NodeFields["Recv"].NodeListFields["List"] {
				/*
					if recv.NodeFields["Type"].Type == "*ast.Ident" {
						recvTypes = append(recvTypes, recv.NodeFields["Type"].StringFields["Name"])
						continue
					}
					if recv.NodeFields["Type"].Type == "*ast.StarExpr" && recv.NodeFields["Type"].NodeFields["X"].Type == "*ast.Ident" {
						recvTypes = append(recvTypes, "*"+recv.NodeFields["Type"].NodeFields["X"].StringFields["Name"])
						continue
					}
				*/
				recvType, err := util.FprintToString(taskCtx.FileSet, recv.NodeFields["Type"].Node)
				if err != nil {
					log.Printf("GetFileInfo FprintToStringFail, err:%+v, recv:%+v", err, util.JsonString(recv))
					continue
				}
				recvTypes = append(recvTypes, recvType)
			}
		}
		recvTypesStr := strings.Join(recvTypes, ",")
		fileInfo.FuncMap[model.FuncKey{
			RecvTypes: recvTypesStr,
			Name:      funcName,
		}] = decl
	}
}

func GetFileInfoImportMap(taskCtx *model.TaskCtx, fileInfo *model.FileInfo) {
	imports, ok := fileInfo.NodeInfo.NodeListFields["Imports"]
	if !ok {
		return
	}
	fileInfo.ImportMap = make(map[string]string)
	for _, imp := range imports {
		if imp.NodeFields["Name"] != nil {
			fileInfo.ImportMap[imp.NodeFields["Name"].StringFields["Name"]] = util.RemoveQuotesIfPresent(imp.NodeFields["Path"].StringFields["Value"])
			continue
		}
		path := util.RemoveQuotesIfPresent(imp.NodeFields["Path"].StringFields["Value"])
		name := util.GetPackageNameFromPath(path)
		fileInfo.ImportMap[name] = path
	}
	for _, imp := range taskCtx.Input.FuncTask.ExtraImports {
		name := ""
		path := imp
		parts := strings.Split(imp, "|")
		if len(parts) == 2 {
			name = parts[0]
			path = parts[1]
		}
		if name == "" {
			name = util.GetPackageNameFromPath(path)
		}
		fileInfo.ImportMap[name] = path
	}
	log.Printf("GetFileInfoImportMap ImportMap:%+v", fileInfo.ImportMap)
}
