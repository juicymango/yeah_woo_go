package logic

import (
	"fmt"
	"log"
	"path/filepath"
	"slices"

	"github.com/juicymango/yeah_woo_go/model"
	"github.com/juicymango/yeah_woo_go/util"
)

// FilterRelevantCallExpr only modify relevant. additionally checking the called function. use the common strategy to recursively check the arguments.
func FilterRelevantCallExpr(taskCtx *model.TaskCtx, nodeInfo *model.NodeInfo) {
	if nodeInfo == nil {
		return
	}
	if nodeInfo.Type != "*ast.CallExpr" {
		return
	}

	fun := nodeInfo.NodeFields["Fun"]
	if fun == nil {
		log.Printf("FilterRelevantCallExpr FunNil, NodeInfo:%s", util.JsonString(nodeInfo))
		return
	}

	if fun.Type == "*ast.Ident" {
		FilterRelevantCallExprLocalFunc(taskCtx, nodeInfo, fun)
		return
		// TODO: if it is a variable
	}

	if fun.Type == "*ast.SelectorExpr" {
		if FilterRelevantCallExprOtherFunc(taskCtx, nodeInfo, fun) {
			return
		}
		FilterRelevantCallExprMethod(taskCtx, nodeInfo, fun)
		return
	}
}

func FilterRelevantCallExprLocalFunc(taskCtx *model.TaskCtx, nodeInfo *model.NodeInfo, fun *model.NodeInfo) {
	dir := filepath.Dir(taskCtx.Input.FuncTask.Source)
	funcName := fun.StringFields["Name"]
	FilterRelevantCallExprFunc(taskCtx, nodeInfo, dir, funcName)
}

func FilterRelevantCallExprFunc(taskCtx *model.TaskCtx, nodeInfo *model.NodeInfo, dir string, funcName string) {
	targetString := fmt.Sprintf("func %s(", funcName)
	targetFilePaths, err := util.Grep(dir, targetString)
	if err != nil {
		log.Printf("FilterRelevantCallExpr GrepErr, err:%+v", err)
		return
	}
	if len(targetFilePaths) == 0 {
		return
	}
	log.Printf("FilterRelevantCallExpr GrepResult, dir:%s, targetString:%s, targetFilePaths:%+v", dir, targetString, targetFilePaths)

	currentFuncTask := taskCtx.Input.FuncTask
	taskCtx.Input.FuncTask.FuncName = funcName
	for _, filePath := range targetFilePaths {
		// new FuncTask
		taskCtx.Input.FuncTask.Source = filePath
		funcNodeInfo := GetFuncNodeInfo(taskCtx)
		relevantFieldNames := GetRelevantFuncFieldNames(taskCtx, nodeInfo, funcNodeInfo)
		if len(relevantFieldNames) > 0 {
			taskCtx.Input.FuncTask.VarNames = slices.Clone(currentFuncTask.VarNames)
			for _, name := range relevantFieldNames {
				if !slices.Contains(taskCtx.Input.FuncTask.VarNames, name) {
					taskCtx.Input.FuncTask.VarNames = append(taskCtx.Input.FuncTask.VarNames, name)
				}
			}
		}
		log.Printf("FilterRelevantCallExpr GrepResult, dir:%s, targetString:%s, targetFilePaths:%+v", dir, targetString, targetFilePaths)

		// FuncTaskMap
		funcTaskKey := util.GetFuncTaskKey(taskCtx.Input.FuncTask)
		if !IsNewFuncTask(taskCtx, funcTaskKey) {
			continue
		}

		// filter
		newFuncNodeInfo := FilterRelevantNodeInfo(taskCtx, funcNodeInfo)
		if newFuncNodeInfo != nil && newFuncNodeInfo.RelevantTaskResult != nil && nodeInfo.RelevantTaskResult != nil {
			log.Printf("FilterRelevantCallExpr NewTaskResult, task:%s, result:%s", util.JsonString(taskCtx.Input.FuncTask), util.JsonString(newFuncNodeInfo.RelevantTaskResult))
			if newFuncNodeInfo.RelevantTaskResult.IsRelevant {
				nodeInfo.RelevantTaskResult.IsRelevant = true
				taskCtx.Input.Funcs = append(taskCtx.Input.Funcs, taskCtx.Input.FuncTask)
			}
		}
		taskCtx.FuncTaskMap[funcTaskKey] = newFuncNodeInfo
	}
	taskCtx.Input.FuncTask = currentFuncTask
}

func GetRelevantFuncFieldNames(taskCtx *model.TaskCtx, nodeInfo *model.NodeInfo, funcNodeInfo *model.NodeInfo) []string {
	if nodeInfo == nil || funcNodeInfo == nil {
		return nil
	}
	varNames := make([]string, 0)
	fieldIdx := 0
	for _, typeFields := range funcNodeInfo.NodeFields["Type"].NodeFields["Params"].NodeListFields["List"] {
		for _, field := range typeFields.NodeListFields["Names"] {
			if fieldIdx < len(nodeInfo.NodeListFields["Args"]) && nodeInfo.NodeListFields["Args"][fieldIdx].RelevantTaskResult != nil && nodeInfo.NodeListFields["Args"][fieldIdx].RelevantTaskResult.IsRelevant && field.Type == "*ast.Ident" {
				varNames = append(varNames, field.StringFields["Name"])
			}
			fieldIdx++
		}
	}
	return varNames
}

func IsNewFuncTask(taskCtx *model.TaskCtx, funcTaskKey model.FuncTaskKey) bool {
	_, ok := taskCtx.FuncTaskMap[funcTaskKey]
	if ok {
		log.Printf("IsNewFuncTask Exists FuncTask, FuncTask:%+v", util.JsonString(taskCtx.Input.FuncTask))
		return false
	}
	if taskCtx.FuncTaskMap == nil {
		taskCtx.FuncTaskMap = make(map[model.FuncTaskKey]*model.NodeInfo)
	}
	taskCtx.FuncTaskMap[funcTaskKey] = nil
	log.Printf("IsNewFuncTask New FuncTask, FuncTask:%+v", util.JsonString(taskCtx.Input.FuncTask))
	return true
}

func FilterRelevantCallExprOtherFunc(taskCtx *model.TaskCtx, nodeInfo *model.NodeInfo, fun *model.NodeInfo) bool {
	if fun.NodeFields["X"].Type != "*ast.Ident" {
		return false
	}
	prefix := fun.NodeFields["X"].StringFields["Name"]
	fileInfo := GetFileInfo(taskCtx)
	if fileInfo == nil {
		log.Printf("FilterRelevantCallExprOtherFunc fileInfo nil, FuncTask:%+v", util.JsonString(taskCtx.Input.FuncTask))
		return false
	}
	importPath := fileInfo.ImportMap[prefix]
	if importPath == "" {
		return false
	}
	dir, err := util.GetAbsoluteImportPath(importPath)
	if err != nil {
		log.Printf("FilterRelevantCallExprOtherFunc GetAbsoluteImportPath fail, importPath:%+v", importPath)
		return false
	}
	funcName := fun.NodeFields["Sel"].StringFields["Name"]
	FilterRelevantCallExprFunc(taskCtx, nodeInfo, dir, funcName)
	return true
}

func FilterRelevantCallExprMethod(taskCtx *model.TaskCtx, nodeInfo *model.NodeInfo, fun *model.NodeInfo) {
}
