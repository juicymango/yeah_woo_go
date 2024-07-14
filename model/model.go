package model

import (
	"go/ast"
	"go/token"
)

type Input struct {
	Method   string     `json:"method"`
	FuncTask FuncTask   `json:"func_task"`
	Funcs    []FuncTask `json:"funcs"`
}

type FuncTask struct {
	Source       string   `json:"source"`
	FuncName     string   `json:"func_name"`
	VarNames     []string `json:"var_names"`
	ShowReturn   bool     `json:"show_return"`
	ShowBreak    bool     `json:"show_break"`
	ShowContinue bool     `json:"show_continue"`
}

type FuncTaskKey struct {
	Source       string `json:"source"`
	FuncName     string `json:"func_name"`
	VarNames     string `json:"var_names"`
	ShowReturn   bool   `json:"show_return"`
	ShowBreak    bool   `json:"show_break"`
	ShowContinue bool   `json:"show_continue"`
}

type TaskCtx struct {
	Input       *Input
	FuncTaskMap map[FuncTaskKey]*NodeInfo
	FileSet     *token.FileSet
}

type NodeInfo struct {
	Node               ast.Node               `json:"-"`
	Type               string                 `json:"type"`
	NodeFields         map[string]*NodeInfo   `json:"node_fields,omitempty"`
	NodeListFields     map[string][]*NodeInfo `json:"node_list_fields,omitempty"`
	StringFields       map[string]string      `json:"string_fields,omitempty"`
	TokenFields        map[string]string      `json:"token_fields,omitempty"`
	RelevantTaskResult *RelevantTaskResult    `json:"-"`
}

type RelevantTaskResult struct {
	IsRelevant       bool
	NotFilterByBlock bool
}
