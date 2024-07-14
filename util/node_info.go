package util

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"maps"
	"reflect"
	"slices"

	"github.com/juicymango/yeah_woo_go/model"
)

func GetNodeInfo(nodeNode ast.Node) *model.NodeInfo {
	nodeInfo := &model.NodeInfo{
		Node:           nodeNode,
		Type:           reflect.TypeOf(nodeNode).String(),
		NodeFields:     make(map[string]*model.NodeInfo),
		NodeListFields: make(map[string][]*model.NodeInfo),
		StringFields:   make(map[string]string),
		TokenFields:    make(map[string]string),
	}

	nodeVal := reflect.ValueOf(nodeNode).Elem()
	for i := 0; i < nodeVal.NumField(); i++ {
		field := nodeVal.Field(i)
		fieldType := field.Type()
		fieldName := nodeVal.Type().Field(i).Name

		// Check if field can be converted to ast.Node
		if fieldType.Implements(reflect.TypeOf((*ast.Node)(nil)).Elem()) {
			if astNode, ok := field.Interface().(ast.Node); ok && !field.IsNil() {
				nodeInfo.NodeFields[fieldName] = GetNodeInfo(astNode)
			}
		} else if fieldType.Kind() == reflect.Slice { // Check if field is a slice
			sliceType := fieldType.Elem()
			if sliceType.Implements(reflect.TypeOf((*ast.Node)(nil)).Elem()) {
				sliceLen := field.Len()
				nodeSlice := make([]*model.NodeInfo, 0, sliceLen)
				for j := 0; j < sliceLen; j++ {
					sliceElement := field.Index(j)
					if astNode, ok := sliceElement.Interface().(ast.Node); ok && !sliceElement.IsNil() {
						nodeSlice = append(nodeSlice, GetNodeInfo(astNode))
					}
				}
				if len(nodeSlice) > 0 {
					nodeInfo.NodeListFields[fieldName] = nodeSlice
				}
			}
		} else if fieldType == reflect.TypeOf("") {
			if s, ok := field.Interface().(string); ok {
				nodeInfo.StringFields[fieldName] = s
			}
		} else if fieldType == reflect.TypeOf(token.Token(0)) {
			if t, ok := field.Interface().(token.Token); ok {
				nodeInfo.TokenFields[fieldName] = t.String()
			}
		}
	}

	return nodeInfo
}

// SetValueToFieldByName sets a value to the named field of a struct.
// It supports setting a field of type []*A from a value of type []interface{}.
// It takes a pointer to the struct 's', the name of the field 'fieldName', and the value 'val' to be assigned.
// It returns an error if the operation cannot be completed.
func SetValueToFieldByName(s interface{}, fieldName string, val interface{}) error {
	rv := reflect.ValueOf(s)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("s must be a pointer to a struct")
	}

	rv = rv.Elem() // Dereference the pointer to get the struct

	field := rv.FieldByName(fieldName)
	if !field.IsValid() {
		return fmt.Errorf("no such field: %s in struct", fieldName)
	}
	if !field.CanSet() {
		return fmt.Errorf("cannot set field %s", fieldName)
	}

	valReflected := reflect.ValueOf(val)
	if valReflected.Kind() == reflect.Slice && field.Kind() == reflect.Slice {
		elemType := field.Type().Elem() // Type that the slice should contain
		slice := reflect.MakeSlice(field.Type(), valReflected.Len(), valReflected.Cap())
		for i := 0; i < valReflected.Len(); i++ {
			elem := valReflected.Index(i)
			if !elem.Type().AssignableTo(elemType) {
				convertedElem := elem.Elem()
				if !convertedElem.Type().AssignableTo(elemType) {
					return fmt.Errorf("element type %s cannot be assigned to slice element type %s", elem.Type(), elemType)
				}
				slice.Index(i).Set(convertedElem)
			} else {
				slice.Index(i).Set(elem)
			}
		}
		field.Set(slice)
	} else if !valReflected.Type().AssignableTo(field.Type()) {
		return fmt.Errorf("provided value type %s does not match field type %s", valReflected.Type(), field.Type())
	} else {
		field.Set(valReflected)
	}

	return nil
}

// CloneNode creates a shallow copy of the provided ast.Node and returns it.
// This function assumes that n is not nil.
func CloneNode(n ast.Node) ast.Node {
	nodeType := reflect.TypeOf(n)
	if nodeType == nil {
		return nil
	}

	// Create a new instance of the underlying type of n.
	newNodePtr := reflect.New(nodeType.Elem())
	newNode := newNodePtr.Interface().(ast.Node)

	// Shallow copy the exported fields from n to the new node.
	originalValue := reflect.ValueOf(n).Elem()
	newValue := newNodePtr.Elem()
	for i := 0; i < originalValue.NumField(); i++ {
		field := originalValue.Field(i)
		if field.CanSet() {
			newValue.Field(i).Set(field)
		}
	}

	return newNode
}

func NodeInfoUpdateNode(nodeInfo *model.NodeInfo) {
	nodeInfo.Node = CloneNode(nodeInfo.Node)
	for name, field := range nodeInfo.NodeFields {
		NodeInfoUpdateNode(field)
		err := SetValueToFieldByName(nodeInfo.Node, name, field.Node)
		if err != nil {
			log.Printf("NodeInfoUpdateNode SetValueToFieldByNameFail NodeType %s, FieldName %s, Err %v", nodeInfo.Type, name, err)
		}
	}
	for name, fields := range nodeInfo.NodeListFields {
		nodes := make([]ast.Node, 0, len(fields))
		for _, field := range fields {
			NodeInfoUpdateNode(field)
			nodes = append(nodes, field.Node)
		}
		err := SetValueToFieldByName(nodeInfo.Node, name, nodes)
		if err != nil {
			log.Printf("NodeInfoUpdateNode SetValueToFieldByNameFail NodeType %s, FieldName %s, Err %v", nodeInfo.Type, name, err)
		}
	}
}

func CloneNodeInfo(nodeInfo *model.NodeInfo) *model.NodeInfo {
	if nodeInfo == nil {
		return nil
	}
	newNodeListFields := make(map[string][]*model.NodeInfo, len(nodeInfo.NodeListFields))
	for name, nodes := range nodeInfo.NodeListFields {
		newNodeListFields[name] = slices.Clone(nodes)
	}
	newNodeInfo := &model.NodeInfo{
		Node:           nodeInfo.Node,
		Type:           nodeInfo.Type,
		NodeFields:     maps.Clone(nodeInfo.NodeFields),
		NodeListFields: newNodeListFields,
		StringFields:   maps.Clone(nodeInfo.StringFields),
		TokenFields:    maps.Clone(nodeInfo.TokenFields),
	}
	return newNodeInfo
}
