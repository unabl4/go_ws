package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
)

var (
	paramsPrefix = "p"             // something we agree on
	tagPrefix    = "`apivalidator" // has to be with the tick
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
	Name            string      // name of the field (raw, original name)
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
	Render(Field) string // display
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

func (v PresenceValidator) Render(f Field) string {
	if f.Type == "string" {
		return fmt.Sprintf("if len(%s.%s) <= 0 {\n\treturn fmt.Errorf(\"%s must me not empty\")\n}", paramsPrefix, f.Name, f.srcQueryParam())
	} else if f.Type == "int" {
		return fmt.Sprintf("if %s.%s == 0 {\n\treturn fmt.Errorf(\"%s must me not empty\")\n}", paramsPrefix, f.Name, f.srcQueryParam())
	} else {
		// should NOT happen
		panic("unsupported type")
	}
}

func (v EnumValidator) Render(f Field) string {
	return ""
}

func (v MinValidator) Render(f Field) string {
	return ""
}

func (v MaxValidator) Render(f Field) string {
	return ""
}

// ---

func parseField(fieldName string, fieldType string, fieldTag string) Field {
	tagKeys := strings.Split(fieldTag, ",")

	f := Field{}
	f.Name = fieldName
	f.Type = fieldType

	for _, tagKey := range tagKeys { // process individual tag keys
		split := strings.Split(tagKey, "=")

		if split[0] == "required" {
			f.Validators = append(f.Validators, PresenceValidator{})
		} else if split[0] == "enum" {
			vals := strings.Split(split[1], "|")
			f.Validators = append(f.Validators, EnumValidator{vals})
		} else if split[0] == "default" {
			// set default value
			if fieldType == "string" {
				f.DefaultValue = split[1]
			} else if fieldType == "int" {
				f.DefaultValue, _ = strconv.Atoi(split[1])
			} else {
				panic("unsupported type")
			}
		} else if split[0] == "paramname" {
			f.SourceParamName = split[1]
		} else if split[0] == "min" {
			i, _ := strconv.Atoi(split[1]) // str -> int
			f.Validators = append(f.Validators, MinValidator{i})
		} else if split[0] == "max" {
			i, _ := strconv.Atoi(split[1]) // str -> int
			f.Validators = append(f.Validators, MaxValidator{i})
		}
	}

	return f
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

				// fmt.Println("Struct parsing >", typeSpec.Name)
				// fmt.Println(typeSpec.Type)
				for _, field := range currStruct.Fields.List {
					var fieldType, tagDescriptor string
					switch field.Type.(type) {
					case *ast.Ident:
						fieldType = field.Type.(*ast.Ident).Name
					}
					// hopefully, other cases will not appear

					if field.Tag != nil {
						tagReflect := reflect.StructTag(field.Tag.Value)
						tagDescriptor = tagReflect.Get(tagPrefix)
					}
					fl := parseField(field.Names[0].Name, fieldType, tagDescriptor)
					fmt.Println(fl)
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

			// fmt.Println(f.Name, "/function params:")
			for _, param := range f.Type.Params.List[1:] {
				fmt.Println(param.Type)
			}

			for _, recv := range f.Recv.List {
				// fmt.Printf("recv type : %T", recv.Type)

				switch xv := recv.Type.(type) {
				case *ast.StarExpr:
					if _, ok := xv.X.(*ast.Ident); ok {
						// fmt.Println("*", si.Name)
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
