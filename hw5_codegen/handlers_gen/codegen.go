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
	muxPrefix    = "srv"
	paramsPrefix = "params"        // something we agree on (does not really matter as long as it is consistent)
	tagPrefix    = "`apivalidator" // has to be with the 'tick'
)

type ApiEndpoint struct {
	Path   string // url path
	Auth   bool
	Method string

	Router // routing description
}

func (e ApiEndpoint) handlerFuncName() string {
	return "handler" + e.Name // Router.Name
}

// ---

func parseApiEndpoint(apiEndpointDescriptor string) ApiEndpoint {
	e := ApiEndpoint{}
	e.Method = "GET" // default method value
	json.Unmarshal([]byte(apiEndpointDescriptor), &e)
	return e
}

// ---

type Router struct { // mux
	Name   string   // raw name
	Params []Params // one or more
}

// ---

type Params struct {
	Name   string // name of the struct type (?)
	Fields []Field
}

type Field struct {
	Name            string      // name of the field (raw, original name)
	Type            string      // 'int' or 'string'
	DefaultValue    interface{} // interface (NULLABLE)
	Validators      []Validator // list of validators
	SourceParamName string      // source field to load from
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
		return fmt.Sprintf("if len(%s.%s) <= 0 {\n\treturn fmt.Errorf(\"%s must me not empty\")\n}", muxPrefix, f.Name, f.srcQueryParam())
	} else if f.Type == "int" {
		return fmt.Sprintf("if %s.%s == 0 {\n\treturn fmt.Errorf(\"%s must me not empty\")\n}", muxPrefix, f.Name, f.srcQueryParam())
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

// stringer
func (v PresenceValidator) String() string {
	return fmt.Sprintf("Must be present")
}

func (v EnumValidator) String() string {
	return fmt.Sprintf("Must be one of: %s", strings.Join(v.AcceptedValues, ","))
}

func (v MinValidator) String() string {
	return fmt.Sprintf("Min=%d", v.Value)
}

func (v MaxValidator) String() string {
	return fmt.Sprintf("Max=%d", v.Value)
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

	var routers = make(map[string][]ApiEndpoint)
	var paramStructs = make(map[string]Params) // e.g "CreateParams" -> [F1,F2,...,Fn], where 'F' = Field 'instance'

	for _, node := range nodeSet.Decls { // all declarations
		g, ok := node.(*ast.GenDecl)
		if ok {
			for _, spec := range g.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				// we are looking for structure types
				currentStruct, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				structName := typeSpec.Name.Name // params struct name

				var fields []Field
				for _, field := range currentStruct.Fields.List {
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
					field := parseField(field.Names[0].Name, fieldType, tagDescriptor)
					fields = append(fields, field)
				}

				params := Params{structName, fields}
				paramStructs[structName] = params
			}

			continue // if GenDecl -> not a FuncDecl (not really necessary)
		}

		// ---
		// FUNCTION (ENDPOINTS) PARSING

		f, ok := node.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if f.Doc == nil {
			continue // there must be a comment
		}

		var params []Params
		for _, param := range f.Type.Params.List[1:] {
			// we deliberately ignore the context param which comes first
			argName := param.Type.(*ast.Ident).Name // Expr -> Ident
			params = append(params, paramStructs[argName])
		}

		var routerName string
		for _, recv := range f.Recv.List {
			// fmt.Printf("recv type : %T", recv.Type)

			switch recvType := recv.Type.(type) {
			case *ast.StarExpr:
				routerName = "*" + recvType.X.(*ast.Ident).Name // compose the name
			case *ast.Ident:
				// not this time anyway, but for sake of compatibility
				routerName = recvType.Name
			}
		}

		// fmt.Println(routerName)

		var apiEndpoint ApiEndpoint
		for _, comment := range f.Doc.List {
			// the function signature comment line is expected in the very first 'Doc'
			if !strings.HasPrefix(comment.Text, "// apigen:api") {
				continue // ignore
			}

			apiEndpoint = parseApiEndpoint(comment.Text[14:])
		}

		apiEndpoint.Name = f.Name.Name
		apiEndpoint.Params = params
		routers[routerName] = append(routers[routerName], apiEndpoint)
	}

	// END OF PARSING
	// THE ACTUAL CODE GENERATION STEP

	for k, v := range routers {
		// multiplexor (mux)
		// fmt.Println("K=", k,v)
		for _, e := range v {
			// mux api endpoints
			paramsName := e.Params[0].Name // ?

			fmt.Printf("func (%s %s) %s(w http.ResponseWriter, r *http.Request) {\n", muxPrefix, k, e.handlerFuncName())
			fmt.Println("\tctx := r.Context()") // context is always present as the first argument
			fmt.Println("\tquery := r.URL.Query()")
			fmt.Println() // newline spacer
			fmt.Printf("\tparams := %s{}\n", paramsName)

			for _, p := range e.Params {
				// fmt.Println(p)

				for _, f := range p.Fields {
					if f.Type == "string" {
						fmt.Printf("\t%s.%s = query.Get(\"%s\")\n", "params", f.Name, f.srcQueryParam())
					} else {
						// extra logic for 'int' handling
						rawVarName := "raw" + f.Name
						intVarName := strings.ToLower(f.Name) + "Int" // already converted
						fmt.Printf("\t%s = query.Get(\"%s\")\n", rawVarName, f.srcQueryParam())
						fmt.Printf("\tif len(%s) > 0 {\n", rawVarName)
						fmt.Printf("\t\t%s, err := strconv.Atoi(%s)\n", intVarName, rawVarName)
						fmt.Println("\t\tif err != nil {")
						fmt.Printf("\t\t\tthrowBadRequest(w, \"%s must be integer\")\n", strings.ToLower(f.Name))
						fmt.Println("\t\t\treturn")
						fmt.Println("\t\t}")

						fmt.Println("")
						fmt.Printf("\t\tparams.%s = %s\n", f.Name, intVarName)
						fmt.Println("\t}")
					}
				}

				fmt.Println()

				// set default values
				for _, f := range p.Fields {
					if f.DefaultValue != nil {
						var zv interface{} // zero value
						if f.Type == "string" {
							zv = `""`
						} else if f.Type == "int" {
							zv = 0
						} else {
							panic("unsupported type")
						}
						fmt.Printf("\tif %s.%s == %s {\n", "params", f.Name, zv)
						fmt.Printf("\t\t%s.%s = \"%s\"\n", "params", f.Name, f.DefaultValue)
						fmt.Println("\t}")
					}
				}

				fmt.Println("")
				fmt.Println("\terr := params.Validate()")
				fmt.Println("\tif err != nil {")
				fmt.Println("\t\tthrowBadRequest(w, err.Error())")
				fmt.Println("\t\treturn")
				fmt.Println("\t}")

				fmt.Println()
				fmt.Println("\tsrvResponse, err := srv.Create(ctx, params)") // ?
				fmt.Println("\tvar ar apiResponse")
				fmt.Println("\tif err != nil {")
				fmt.Println("\t\te := err.(ApiError)")
				fmt.Println("\t\tw.WriteHeader(e.HTTPStatus)")
				fmt.Println("\t\tar = apiResponse{e.Err.Error(), nil}")
				fmt.Println("\t} else {")
				fmt.Println("\t\tar = apiResponse{\"\", srvResponse}")
				fmt.Println("\t}")

				fmt.Println()
				fmt.Println("\tj, err := encodeJson(ar)")
				fmt.Println("\tif err != nil {")
				fmt.Println("\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)")
				fmt.Println("\t\treturn")
				fmt.Println("\t}")
				fmt.Println("\tw.Write(j)")
				fmt.Println("}")
			}
		}
	}
}
