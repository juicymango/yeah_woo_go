package logic

import (
	"fmt"
	"log"
	"path/filepath"
	"slices"
	"maps"

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
	currentResult := GetFuncTaskResult(taskCtx)
	for _, filePath := range targetFilePaths {
		taskCtx.Input.FuncTask = currentFuncTask
		util.SetSubTask(&taskCtx.Input.FuncTask)
		taskCtx.Input.FuncTask.FuncName = funcName
		taskCtx.Input.FuncTask.RecvTypes = receiver
		taskCtx.Input.FuncTask.Source = filePath
		result := GetFuncTaskResult(taskCtx)
		if result.FuncNodeInfo == nil {
			continue
		}
		taskCtx.Input.FuncTask = result.FuncTask
		if taskCtx.Input.FuncTask.FarawayMatch {
			taskCtx.Input.FuncTask.VarNames = currentFuncTask.VarNames
		}

		relevantFieldNames := GetRelevantFuncFieldNames(taskCtx, nodeInfo, result.FuncNodeInfo)
		if taskCtx.Input.FuncTask.OnlyRelevantFunc {
			relevantFieldNames = nil
		}
		if len(relevantFieldNames) > 0 {
			taskCtx.Input.FuncTask.VarNames = util.MergeAndDeduplicate(taskCtx.Input.FuncTask.VarNames, relevantFieldNames)
		}

		log.Printf("FilterRelevantCallExpr GrepResult, dir:%s, targetString:%s, targetFilePaths:%+v", dir, targetString, targetFilePaths)
		if len(taskCtx.Input.FuncTask.VarNames) == 0 && !isFunNameRelevant {
			continue
		}

		if currentResult.CalleeMap == nil {
			currentResult.CalleeMap = make(map[model.FuncTaskKey]*model.FuncTaskResult)
		}
		currentResult.CalleeMap[util.GetFuncTaskKey(result.FuncTask)] = result
		if result.CallerMap == nil {
			result.CallerMap = make(map[model.FuncTaskKey]*model.FuncTaskResult)
		}
		result.CallerMap[util.GetFuncTaskKey(currentResult.FuncTask)] = currentResult

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

func FilterRelevantFuncCallerKey(taskCtx *model.TaskCtx, filePath string, receiver string, funcName string) {
	currentFuncTask := taskCtx.Input.FuncTask
	currentResult := GetFuncTaskResult(taskCtx)
	util.SetSubTask(&taskCtx.Input.FuncTask)
	taskCtx.Input.FuncTask.FuncName = funcName
	taskCtx.Input.FuncTask.RecvTypes = receiver
	taskCtx.Input.FuncTask.Source = filePath
	// new FuncTask
	result := GetFuncTaskResult(taskCtx)
	if result.FuncNodeInfo == nil {
		return
	}
	taskCtx.Input.FuncTask = result.FuncTask
	if currentResult.CallerMap == nil {
		currentResult.CallerMap = make(map[model.FuncTaskKey]*model.FuncTaskResult)
	}
	currentResult.CallerMap[util.GetFuncTaskKey(result.FuncTask)] = result
	if result.CalleeMap == nil {
		result.CalleeMap = make(map[model.FuncTaskKey]*model.FuncTaskResult)
	}
	result.CalleeMap[util.GetFuncTaskKey(currentResult.FuncTask)] = currentResult

	if CheckNeedRunAndMergeVarNames(taskCtx, result) {
		result.FilterRelevantNodeInfo = FilterRelevantNodeInfo(taskCtx, result.FuncNodeInfo)
		log.Printf("FilterRelevantFuncCallerKey NewTaskResult, task:%s, result:%s", util.JsonString(taskCtx.Input.FuncTask), util.JsonString(result.FilterRelevantNodeInfo.RelevantTaskResult))
	}
	if result.FilterRelevantNodeInfo != nil && result.FilterRelevantNodeInfo.RelevantTaskResult != nil {
		result.FilterRelevantNodeInfo.RelevantTaskResult.IsRelevant = true
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

func FilterRelevantFuncCallerKeys(taskCtx *model.TaskCtx, nodeInfo *model.NodeInfo) {
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
	for _, key := range taskCtx.Input.FuncTask.FuncCallerKeys {
		funcTaskKey, err := util.StringToFuncTaskKey(key)
		if err != nil {
			log.Printf("FilterRelevantFuncCallerKeys StringToFuncTaskKey fail, key:%s err:%+v", key, err)
			continue
		}
		FilterRelevantFuncCallerKey(taskCtx, funcTaskKey.Source, funcTaskKey.RecvTypes, funcTaskKey.FuncName)
	}
}

func GenCalleeTree(result *model.FuncTaskResult) {
	result.FuncTask.CalleeTree = GenCalleeTreeSub(result, nil, result.FuncTask.CollectComments)
}

func GenCalleeTreeSub(result *model.FuncTaskResult, hasGenMap map[string]bool, collectComments bool) map[string]interface{} {
	key := util.FuncTaskKeyToString(util.GetFuncTaskKey(result.FuncTask))
	if hasGenMap == nil {
		hasGenMap = make(map[string]bool)
	}
	hasGenMap[key] = true
	tree := make(map[string]interface{}, len(result.CalleeMap)+1)
	if collectComments && len(result.FuncTask.Comments) > 0 {
		tree["comments"] = result.FuncTask.Comments
	}
	for calleeKey, calleeResult := range result.CalleeMap {
		calleeKeyStr := util.FuncTaskKeyToString(calleeKey)
		if hasGenMap[calleeKeyStr] {
			continue
		}
		tree[calleeKeyStr] = GenCalleeTreeSub(calleeResult, maps.Clone(hasGenMap), collectComments)
	}
	return tree
}

func GenCallerTree(result *model.FuncTaskResult) {
	result.FuncTask.CallerTree = GenCallerTreeSub(result, nil, result.FuncTask.CollectComments)
}

func GenCallerTreeSub(result *model.FuncTaskResult, hasGenMap map[string]bool, collectComments bool) map[string]interface{} {
	key := util.FuncTaskKeyToString(util.GetFuncTaskKey(result.FuncTask))
	if hasGenMap == nil {
		hasGenMap = make(map[string]bool)
	}
	hasGenMap[key] = true
	tree := make(map[string]interface{}, len(result.CallerMap)+1)
	if collectComments && len(result.FuncTask.Comments) > 0 {
		tree["comments"] = result.FuncTask.Comments
	}
	for callerKey, callerResult := range result.CallerMap {
		callerKeyStr := util.FuncTaskKeyToString(callerKey)
		if hasGenMap[callerKeyStr] {
			continue
		}
		tree[callerKeyStr] = GenCallerTreeSub(callerResult, maps.Clone(hasGenMap), collectComments)
	}
	return tree
}
