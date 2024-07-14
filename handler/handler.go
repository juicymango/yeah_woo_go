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
	methodFuncMap := map[string]func(string, *model.Input){
		"GetRelevantFuncs": GetRelevantFuncs,
		"GetFuncNodeInfo":  GetFuncNodeInfo,
		"GetFileNodeInfo":  GetFileNodeInfo,
	}
	method := methodFuncMap[input.Method]
	if method == nil {
		log.Fatalf("unknown method, methods:%+v, input:%+v", methodFuncMap, input)
	}
	method(filePath, &input)
}

func GetFuncNodeInfo(filePath string, input *model.Input) {
	// Create a new token file set which is needed for parsing
	fset := token.NewFileSet()

	// Parse the file containing the Go program
	fileNode, err := parser.ParseFile(fset, input.FuncTask.Source, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	funcDecl := util.GetFunc(fileNode, input.FuncTask.FuncName)
	nodeInfo := util.GetNodeInfo(funcDecl)
	funcJson, jsonErr := json.Marshal(nodeInfo)
	if jsonErr != nil {
		log.Printf("GetFuncNodeInfo MarshalErr %+v", jsonErr)
		return
	}
	fmt.Println(string(funcJson))
}

func GetFileNodeInfo(filePath string, input *model.Input) {
	// Create a new token file set which is needed for parsing
	fset := token.NewFileSet()

	// Parse the file containing the Go program
	fileNode, err := parser.ParseFile(fset, input.FuncTask.Source, nil, parser.ParseComments)
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

func GetRelevantFuncs(filePath string, input *model.Input) {
	taskCtx := &model.TaskCtx{
		Input:       input,
		FuncTaskMap: make(map[model.FuncTaskKey]*model.NodeInfo),
		FileSet:     token.NewFileSet(),
	}
	for idx := 0; idx < len(taskCtx.Input.Funcs); idx++ {
		taskCtx.Input.FuncTask = taskCtx.Input.Funcs[idx]
		funcTaskKey := util.GetFuncTaskKey(taskCtx.Input.FuncTask)

		if logic.IsNewFuncTask(taskCtx, funcTaskKey) {
			funcNodeInfo := logic.GetFuncNodeInfo(taskCtx)
			newFuncNodeInfo := logic.FilterRelevantNodeInfo(taskCtx, funcNodeInfo)
			taskCtx.FuncTaskMap[funcTaskKey] = newFuncNodeInfo
		}

		funcNodeInfo := taskCtx.FuncTaskMap[funcTaskKey]
		if funcNodeInfo == nil {
			log.Printf("GetRelevantFuncs FuncTaskMapNotFound %+v", util.JsonString(&funcTaskKey))
			continue
		}
		util.NodeInfoUpdateNode(funcNodeInfo)
		fmt.Printf("//file://%s\n", input.FuncTask.Source)
		err := printer.Fprint(os.Stdout, taskCtx.FileSet, funcNodeInfo.Node)
		if err != nil {
			log.Printf("GetRelevantFuncs FprintErr %+v", err)
			return
		}
		fmt.Println()
		fmt.Println()
	}

	formattedJSON, err := FormatJSONObject(taskCtx.Input)
	if err != nil {
		log.Fatal(err)
	}

	// Write the formatted JSON to file
	err = WriteToFile(filePath, formattedJSON)
	if err != nil {
		log.Fatal(err)
	}
}

// FormatJSONObject takes an interface{} object, marshals it into JSON, and formats it.
func FormatJSONObject(obj interface{}) (string, error) {
	formattedJSON, err := json.MarshalIndent(obj, "", "    ") // 4 spaces for indentation
	if err != nil {
		return "", err
	}
	return string(formattedJSON), nil
}

// WriteToFile writes a string to a file.
func WriteToFile(filename string, data string) error {
	// Create a file or overwrite if it already exists
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the data to the file
	_, err = file.WriteString(data)
	if err != nil {
		return err
	}

	return nil
}
