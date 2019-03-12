package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
)

type ApiEntry struct {
	Url    string
	Auth   bool
	Method string
}

// ---

func parseApiEntry(apiEntryDescriptor string) ApiEntry {
	ae := ApiEntry{}
	ae.Method = "GET" // default method value
	json.Unmarshal([]byte(apiEntryDescriptor), &ae)
	return ae
}

// ---

type Field struct {
	Name            string      // name of the field (raw name)
	Type            string      // 'int' or 'string'
	DefaultValue    interface{} // interface (NULLABLE)
	Validators      []Validator // list of validators
	SourceParamName string      // source field to load from
}

type Struct struct {
	Fields []Field
}

// name of the query param to load from
func (f Field) srcQueryParam() string {
	if f.SourceParamName != "" { // override?
		return f.SourceParamName
	}

	return strings.ToLower(f.Name) // lowercase (by definition)
}

// ---

type Validator interface {
	IsValid(input interface{}) bool
	// RenderError(fieldName string) string
}

type PresenceValidator struct{} // empty (no fields needed)

type MinValidator struct {
	Value int
}

type MaxValidator struct {
	Value int
}

type EnumValidator struct {
	AcceptedValues []string
}

func (v PresenceValidator) IsValid(input interface{}) bool {
	if s, ok := input.(string); ok { // string
		return len(s) > 0 // <> "" (not equal to an empty string)
	}

	if i, ok := input.(int); ok { // int
		return i != 0 // by definition
	}

	// supposedly, ref types (pointers, maps, etc)
	return input != nil
}

func (v MinValidator) IsValid(input interface{}) bool {
	if s, ok := input.(string); ok { // string
		return len(s) >= v.Value
	}

	if i, ok := input.(int); ok { // int
		return i >= v.Value
	}

	return true
}

func (v MaxValidator) IsValid(input interface{}) bool {
	if s, ok := input.(string); ok { // string
		return len(s) <= v.Value
	}

	if i, ok := input.(int); ok { // int
		return i <= v.Value
	}

	return true
}

func (v EnumValidator) IsValid(input interface{}) bool {
	for _, v := range v.AcceptedValues {
		if v == input {
			return true // item found -> valid
		}
	}

	return false
}

// ===

// код писать тут
func main() {
	fset := token.NewFileSet()
	nodeSet, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	// out, _ := os.Create(os.Args[2]) // create the file

	// write
	// fmt.Fprintln(out, `package `+f.Name.Name) // package is required
	// fmt.Fprintln(out)                            // empty line
	// fmt.Fprintln(out, `import "encoding/binary"`)
	// fmt.Fprintln(out, `import "bytes"`)
	// fmt.Fprintln(out) // empty line

	for _, node := range nodeSet.Decls { // all declarations
		// fmt.Println(node)

		g, ok := node.(*ast.GenDecl)
		if ok {
			// fmt.Printf("SKIP %v is not *ast.GenDecl\n", f)
			for _, spec := range g.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					// fmt.Printf("SKIP %T is not ast.TypeSpec\n", spec)
					continue
				}

				currStruct, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					fmt.Printf("SKIP %T is not ast.StructType\n", currStruct)
					continue
				}

				fmt.Println("Struct parsing >", typeSpec.Name)
				fmt.Println(typeSpec.Type)
				for _, e := range currStruct.Fields.List {
					fmt.Println(e.Type)
					if e.Tag != nil {
						fmt.Println(e.Tag.Value)
					}
				}
			}

			fmt.Println("---")
			continue // if GenDecl -> not a FuncDecl
		}

		// ---

		f, ok := node.(*ast.FuncDecl)
		if !ok {
			// fmt.Printf("SKIP %v is not *ast.GenDecl\n", f)
			continue
		} else {
			// fmt.Println(f)
			if f.Doc == nil {
				continue
			}

			fmt.Println(f.Name, "/function params:")
			for _, param := range f.Type.Params.List[1:] {
				fmt.Println(param.Type)
			}

			for _, recv := range f.Recv.List {
				fmt.Printf("recv type : %T", recv.Type)

				switch xv := recv.Type.(type) {
				case *ast.StarExpr:
					if si, ok := xv.X.(*ast.Ident); ok {
						fmt.Println("*", si.Name)
					}
				case *ast.Ident:
					// not this time anyway
					fmt.Println(xv.Name)
				}
			}

			for _, comment := range f.Doc.List {
				if !strings.HasPrefix(comment.Text, "// apigen:api") {
					continue // ignore
				}

				apiEntryDescriptor := parseApiEntry(comment.Text[14:])
				fmt.Println(apiEntryDescriptor)
			}
		}

		fmt.Println("---")
	}
}
