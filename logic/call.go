package logic

import (
	"fmt"
	"log"
	"path/filepath"

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

		result := GetFuncTaskResult(taskCtx)
		relevantFieldNames := GetRelevantFuncFieldNames(taskCtx, nodeInfo, result.FuncNodeInfo)
		if len(relevantFieldNames) > 0 {
			taskCtx.Input.FuncTask.VarNames = util.MergeAndDeduplicate(taskCtx.Input.FuncTask.VarNames, relevantFieldNames)
		}
		log.Printf("FilterRelevantCallExpr GrepResult, dir:%s, targetString:%s, targetFilePaths:%+v", dir, targetString, targetFilePaths)

		if result.FuncNodeInfo == nil {
			continue
		}
		if !CheckNeedRunAndMergeVarNames(taskCtx, result) {
			continue
		}

		// filter
		newFuncNodeInfo := FilterRelevantNodeInfo(taskCtx, result.FuncNodeInfo)
		result.FilterRelevantNodeInfo = newFuncNodeInfo
		if newFuncNodeInfo != nil && newFuncNodeInfo.RelevantTaskResult != nil && nodeInfo.RelevantTaskResult != nil {
			log.Printf("FilterRelevantCallExpr NewTaskResult, task:%s, result:%s", util.JsonString(taskCtx.Input.FuncTask), util.JsonString(newFuncNodeInfo.RelevantTaskResult))
			if newFuncNodeInfo.RelevantTaskResult.IsRelevant {
				nodeInfo.RelevantTaskResult.IsRelevant = true
			}
		}
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

func GetFuncTaskResult(taskCtx *model.TaskCtx) *model.FuncTaskResult {
	if taskCtx.FuncTaskResults == nil {
		taskCtx.FuncTaskResults = make([]*model.FuncTaskResult, 0)
	}
	if taskCtx.FuncTaskMap == nil {
		taskCtx.FuncTaskMap = make(map[model.FuncTaskKey]*model.FuncTaskResult)
	}
	funcTaskKey := util.GetFuncTaskKey(taskCtx.Input.FuncTask)
	result := taskCtx.FuncTaskMap[funcTaskKey]
	if result == nil {
		result = &model.FuncTaskResult{
			FuncNodeInfo: GetFuncNodeInfo(taskCtx),
		}
		if result.FuncNodeInfo == nil {
			log.Printf("GetFuncTaskResult FuncNodeInfoNil, funcTaskKey:%+v", util.JsonString(funcTaskKey))
		}
		taskCtx.FuncTaskMap[funcTaskKey] = result
		taskCtx.FuncTaskResults = append(taskCtx.FuncTaskResults, result)
		log.Printf("GetFuncTaskResult New FuncTask, funcTaskKey:%+v", util.JsonString(funcTaskKey))
	}
	return result
}

func CheckNeedRunAndMergeVarNames(taskCtx *model.TaskCtx, result *model.FuncTaskResult) bool {
	if result.FuncNodeInfo == nil {
		return false
	}
	if !result.Started {
		result.Started = true
		result.FuncTask = taskCtx.Input.FuncTask
		log.Printf("CheckNeedRunAndMergeVarNames NotStarted, FuncTask:%+v", util.JsonString(taskCtx.Input.FuncTask))
		return true
	}
	newVarNames := util.MergeAndDeduplicate(taskCtx.Input.FuncTask.VarNames, result.FuncTask.VarNames)
	if len(newVarNames) > len(result.FuncTask.VarNames) {
		log.Printf("CheckNeedRunAndMergeVarNames NewVarNames, FuncTask:%+v, newVarNames:%v, oldVarNames:%v", util.JsonString(taskCtx.Input.FuncTask), newVarNames, result.FuncTask.VarNames)
		result.FuncTask.VarNames = newVarNames
		taskCtx.Input.FuncTask.VarNames = newVarNames
		result.FilterRelevantNodeInfo = nil
		result.Started = true
		return true
	}
	return false
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
