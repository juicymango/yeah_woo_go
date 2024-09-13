package model

import (
	"go/ast"
	"go/token"
	"time"
)

type Input struct {
	Method   string     `json:"method"`
	FuncTask FuncTask   `json:"-"`
	Funcs    []FuncTask `json:"funcs"`
}

type FuncTask struct {
	Source           string   `json:"source"`
	RecvTypes        string   `json:"recv_types"` // seperated by ","
	FuncName         string   `json:"func_name"`
	VarNames         []string `json:"var_names"`
	ShowReturn       bool     `json:"show_return"`
	ShowBreak        bool     `json:"show_break"`
	ShowContinue     bool     `json:"show_continue"`
	ExactMatch       bool     `json:"exact_match"`
	EnableCall       bool     `json:"enable_call"`
	FarawayMatch     bool     `json:"faraway_match"`
	OnlyRelevantFunc bool     `json:"only_relevant_func"`
}

type FuncTaskKey struct {
	Source    string `json:"source"`
	RecvTypes string `json:"recv_types"`
	FuncName  string `json:"func_name"`
}

type FuncTaskResult struct {
	FuncTask               FuncTask
	FuncNodeInfo           *NodeInfo
	FilterRelevantNodeInfo *NodeInfo
	IsFromInput            bool
	Started                bool
}

type TaskCtx struct {
	Input           *Input
	FuncTaskResults []*FuncTaskResult
	FuncTaskMap     map[FuncTaskKey]*FuncTaskResult
	FileSet         *token.FileSet
	FileInfoMap     map[string]*FileInfo
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

type FileInfo struct {
	NodeInfo  *NodeInfo
	Package   string
	FuncMap   map[FuncKey]*NodeInfo
	ImportMap map[string]string
}

type FuncKey struct {
	RecvTypes string
	Name      string
}

type Metrics struct {
	Count       int           `json:"count"`
	TotalTime   time.Duration `json:"-"`
	TotalTimeMS int64         `json:"total_time_ms"`
	AvgTimeMS   float64       `json:"avg_time_ms"`
}
