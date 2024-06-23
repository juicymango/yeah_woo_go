package handler

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"log"
	"os"

	"github.com/juicymango/yeah_woo_go/logic"
	"github.com/juicymango/yeah_woo_go/model"
	"github.com/juicymango/yeah_woo_go/util"
)

func Handle(filePath string) {
	// Read the JSON file from the provided path.
	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	// Declare a variable to hold the unmarshaled content.
	var input model.Input

	// Unmarshal the JSON data into the Input structure.
	err = json.Unmarshal(fileContent, &input)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
		return
	}
	switch input.Method {
	case "GetRelevantFunc":
		GetRelevantFunc(&input)
	case "GetFuncNodeInfo":
		GetFuncNodeInfo(&input)
	case "GetFileNodeInfo":
		GetFileNodeInfo(&input)
	default:
		log.Fatalf("unknown method: %+v", input)
	}
}

func GetFuncNodeInfo(input *model.Input) {
	// Create a new token file set which is needed for parsing
	fset := token.NewFileSet()

	// Parse the file containing the Go program
	fileNode, err := parser.ParseFile(fset, input.Source, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	funcDecl := util.GetFunc(fileNode, input.FuncName)
	nodeInfo := util.GetNodeInfo(funcDecl)
	funcJson, jsonErr := json.Marshal(nodeInfo)
	if jsonErr != nil {
		log.Printf("GetFuncNodeInfo MarshalErr %+v", err)
		return
	}
	fmt.Println(string(funcJson))
}

func GetFileNodeInfo(input *model.Input) {
	// Create a new token file set which is needed for parsing
	fset := token.NewFileSet()

	// Parse the file containing the Go program
	fileNode, err := parser.ParseFile(fset, input.Source, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	nodeInfo := util.GetNodeInfo(fileNode)
	funcJson, jsonErr := json.Marshal(nodeInfo)
	if jsonErr != nil {
		log.Printf("GetFuncNodeInfo MarshalErr %+v", err)
		return
	}
	fmt.Println(string(funcJson))
}

func GetRelevantFunc(input *model.Input) {
	// Create a new token file set which is needed for parsing
	fset := token.NewFileSet()

	// Parse the file containing the Go program
	fileNode, err := parser.ParseFile(fset, input.Source, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	funcDecl := util.GetFunc(fileNode, input.FuncName)
	nodeInfo := util.GetNodeInfo(funcDecl)
	taskCtx := model.TaskCtx{
		Input: input,
	}
	newNodeInfo, _ := logic.FilterRelevantNodeInfo(&taskCtx, nodeInfo)

	util.NodeInfoUpdateNode(newNodeInfo)
	err = printer.Fprint(os.Stdout, fset, newNodeInfo.Node)
	if err != nil {
		log.Printf("GetRelevantFunc FprintErr %+v", err)
		return
	}
}
