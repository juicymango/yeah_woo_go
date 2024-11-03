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
	FilterRelevantCallExprFunc(taskCtx, nodeInfo, dir, "", funcName, false)
}

// FilterRelevantCallExprFunc TODO: only support single receiver
func FilterRelevantCallExprFunc(taskCtx *model.TaskCtx, nodeInfo *model.NodeInfo, dir string, receiver string, funcName string, isManual bool) {
	isFunNameRelevant := isManual || slices.Contains(taskCtx.Input.FuncTask.VarNames, funcName)
	if !isFunNameRelevant && taskCtx.Input.FuncTask.OnlyRelevantFunc {
		return
	}
	targetString := fmt.Sprintf("func %s(", funcName)
	if receiver != "" {
		targetString = fmt.Sprintf("%s) %s(", receiver, funcName)
	}
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
	util.SetSubTask(&taskCtx.Input.FuncTask)
	taskCtx.Input.FuncTask.FuncName = funcName
	taskCtx.Input.FuncTask.RecvTypes = receiver
	for _, filePath := range targetFilePaths {
		// new FuncTask
		taskCtx.Input.FuncTask.Source = filePath
		result := GetFuncTaskResult(taskCtx)
		if result.FuncNodeInfo == nil {
			continue
		}
		util.MergeFuncTaskFromResult(&taskCtx.Input.FuncTask, result)
		relevantFieldNames := GetRelevantFuncFieldNames(taskCtx, nodeInfo, result.FuncNodeInfo)
		if taskCtx.Input.FuncTask.OnlyRelevantFunc {
			relevantFieldNames = nil
		}
		if !taskCtx.Input.FuncTask.FarawayMatch {
			taskCtx.Input.FuncTask.VarNames = nil
		}
		if len(relevantFieldNames) > 0 {
			taskCtx.Input.FuncTask.VarNames = util.MergeAndDeduplicate(taskCtx.Input.FuncTask.VarNames, relevantFieldNames)
		}
		log.Printf("FilterRelevantCallExpr GrepResult, dir:%s, targetString:%s, targetFilePaths:%+v", dir, targetString, targetFilePaths)
		if len(taskCtx.Input.FuncTask.VarNames) == 0 && !isFunNameRelevant {
			continue
		}
		if CheckNeedRunAndMergeVarNames(taskCtx, result) {
			result.FilterRelevantNodeInfo = FilterRelevantNodeInfo(taskCtx, result.FuncNodeInfo)
			log.Printf("FilterRelevantCallExpr NewTaskResult, task:%s, result:%s", util.JsonString(taskCtx.Input.FuncTask), util.JsonString(result.FilterRelevantNodeInfo.RelevantTaskResult))
		}
		if result.FilterRelevantNodeInfo != nil && result.FilterRelevantNodeInfo.RelevantTaskResult != nil {
			if isFunNameRelevant {
				result.FilterRelevantNodeInfo.RelevantTaskResult.IsRelevant = true
			}
			if result.FilterRelevantNodeInfo.RelevantTaskResult.IsRelevant && nodeInfo != nil && nodeInfo.RelevantTaskResult != nil {
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
			FuncTask:     taskCtx.Input.FuncTask,
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
	// skip std TODO: support e.g. encoding/json
	if importPath == prefix {
		return false
	}
	dir, err := util.GetAbsoluteImportPath(importPath)
	if err != nil {
		log.Printf("FilterRelevantCallExprOtherFunc GetAbsoluteImportPath fail, importPath:%+v", importPath)
		return false
	}
	funcName := fun.NodeFields["Sel"].StringFields["Name"]
	FilterRelevantCallExprFunc(taskCtx, nodeInfo, dir, "", funcName, false)
	return true
}

func FilterRelevantCallExprMethod(taskCtx *model.TaskCtx, nodeInfo *model.NodeInfo, fun *model.NodeInfo) {
}

func FilterRelevantFuncCalls(taskCtx *model.TaskCtx, nodeInfo *model.NodeInfo) {
	if nodeInfo == nil {
		return
	}
	if nodeInfo.Type != "*ast.FuncDecl" {
		return
	}
	fileInfo := GetFileInfo(taskCtx)
	if fileInfo == nil {
		log.Printf("FilterRelevantFuncCalls fileInfo nil, FuncTask:%+v", util.JsonString(taskCtx.Input.FuncTask))
		return
	}
	for _, funcCall := range taskCtx.Input.FuncTask.FuncCalls {
		recv, pkg, funcName, err := util.ParseFuncCall(funcCall)
		if err != nil {
			log.Printf("FilterRelevantFuncCalls ParseFuncCall fail, funcCall:%s err:%+v", funcCall, err)
			continue
		}
		dir := filepath.Dir(taskCtx.Input.FuncTask.Source)
		if pkg != "" {
			importPath := fileInfo.ImportMap[pkg]
			if importPath == "" {
				continue
			}
			dir, err = util.GetAbsoluteImportPath(importPath)
			if err != nil {
				log.Printf("FilterRelevantFuncCalls GetAbsoluteImportPath fail, importPath:%+v", importPath)
				continue
			}
		}
		FilterRelevantCallExprFunc(taskCtx, nil, dir, recv, funcName, true)
	}
}
