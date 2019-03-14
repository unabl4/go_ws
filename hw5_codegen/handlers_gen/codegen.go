package main

import (
	"bytes"
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
	Path   string `json:"url"` // url path
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
	values := []string{}
	for _, e := range v.AcceptedValues {
		values = append(values, fmt.Sprintf("\"%s\"", e))
	}

	foundIndVar := strings.ToLower(f.Name) + "Found" // indicator variable name
	acceptedValues := fmt.Sprintf("[]string{ %s }", strings.Join(values, ", "))
	acceptedValuesError := "[" + strings.Join(v.AcceptedValues, ", ") + "]"
	s := bytes.Buffer{}
	fmt.Fprintf(&s, "%s := false\n", foundIndVar)
	fmt.Fprintf(&s, "for _, v := range %s {\n", acceptedValues)
	fmt.Fprintf(&s, "\tif v == %s.%s {\n\t\t%s = true\n\t\tbreak\n\t}\n", muxPrefix, f.Name, foundIndVar)
	fmt.Fprintln(&s, "}")

	fmt.Fprintf(&s, "if !%s {\n", foundIndVar)
	fmt.Fprintf(&s, "\treturn fmt.Errorf(\"%s must be one of %s\")\n", f.srcQueryParam(), acceptedValuesError)
	fmt.Fprintln(&s, "}")

	return s.String()
}

func (v MinValidator) Render(f Field) string {
	if f.Type == "string" {
		return fmt.Sprintf("if len(%s.%s) < %d {\n\treturn fmt.Errorf(\"%s len must be >= %d\")\n}", muxPrefix, f.Name, v.Value, f.srcQueryParam(), v.Value)
	} else if f.Type == "int" {
		return fmt.Sprintf("if %s.%s < %d {\n\treturn fmt.Errorf(\"%s must be >= %d\")\n}", muxPrefix, f.Name, v.Value, f.srcQueryParam(), v.Value)
	} else {
		// should NOT happen
		panic("unsupported type")
	}
}

func (v MaxValidator) Render(f Field) string {
	if f.Type == "string" {
		return fmt.Sprintf("if len(%s.%s) > %d {\n\treturn fmt.Errorf(\"%s len must be <= %d\")\n}", muxPrefix, f.Name, v.Value, f.srcQueryParam(), v.Value)
	} else if f.Type == "int" {
		return fmt.Sprintf("if %s.%s > %d {\n\treturn fmt.Errorf(\"%s must be <= %d\")\n}", muxPrefix, f.Name, v.Value, f.srcQueryParam(), v.Value)
	} else {
		// should NOT happen
		panic("unsupported type")
	}
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

	out, _ := os.Create(os.Args[2])   // create the file
	fmt.Fprintln(out, `package main`) // ?
	fmt.Fprintln(out)
	fmt.Fprintln(out, `import "bytes"`)
	fmt.Fprintln(out, `import "encoding/json"`)
	fmt.Fprintln(out, `import "fmt"`)
	fmt.Fprintln(out, `import "net/http"`)
	fmt.Fprintln(out, `import "strconv"`)
	fmt.Fprintln(out) // empty line

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
			paramsName := e.Params[0].Name // hack

			fmt.Fprintf(out, "func (%s %s) %s(w http.ResponseWriter, r *http.Request) {\n", muxPrefix, k, e.handlerFuncName())
			fmt.Fprintln(out, "\tctx := r.Context()") // context is always present as the first argument
			fmt.Fprintln(out) // newline spacer
			fmt.Fprintf(out, "\tparams := %s{}\n", paramsName)

			for _, p := range e.Params {
				for _, f := range p.Fields {
					if f.Type == "string" {
						fmt.Fprintf(out, "\t%s.%s = r.FormValue(\"%s\")\n", "params", f.Name, f.srcQueryParam())
					} else {
						// extra logic for 'int' handling
						rawVarName := "raw" + f.Name
						intVarName := strings.ToLower(f.Name) + "Int" // already converted
						fmt.Fprintf(out, "\t%s := r.FormValue(\"%s\")\n", rawVarName, f.srcQueryParam())
						fmt.Fprintf(out, "\tif len(%s) > 0 {\n", rawVarName)
						fmt.Fprintf(out, "\t\t%s, err := strconv.Atoi(%s)\n", intVarName, rawVarName)
						fmt.Fprintln(out, "\t\tif err != nil {")
						fmt.Fprintf(out, "\t\t\tthrowBadRequest(w, \"%s must be int\")\n", strings.ToLower(f.Name))
						fmt.Fprintln(out, "\t\t\treturn")
						fmt.Fprintln(out, "\t\t}")

						fmt.Fprintln(out, "")
						fmt.Fprintf(out, "\t\tparams.%s = %s\n", f.Name, intVarName)
						fmt.Fprintln(out, "\t}")
					}
				}

				fmt.Fprintln(out)

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
						fmt.Fprintf(out, "\tif %s.%s == %s {\n", "params", f.Name, zv)
						fmt.Fprintf(out, "\t\t%s.%s = \"%s\"\n", "params", f.Name, f.DefaultValue)
						fmt.Fprintln(out, "\t}")
					}
				}

				fmt.Fprintln(out, "")
				fmt.Fprintln(out, "\terr := params.Validate()")
				fmt.Fprintln(out, "\tif err != nil {")
				fmt.Fprintln(out, "\t\tthrowBadRequest(w, err.Error())")
				fmt.Fprintln(out, "\t\treturn")
				fmt.Fprintln(out, "\t}")

				fmt.Fprintln(out)
				fmt.Fprintf(out, "\tsrvResponse, err := srv.%s(ctx, params)\n", e.Name) // ?
				fmt.Fprintln(out, "\tvar ar apiResponse")
				fmt.Fprintln(out, "\tif err != nil {")
				fmt.Fprintln(out, "\t\tswitch e := err.(type) {")
				fmt.Fprintln(out, "\t\tcase ApiError:")
				fmt.Fprintln(out, "\t\t\tw.WriteHeader(e.HTTPStatus)")
				fmt.Fprintln(out, "\t\t\tar = apiResponse{e.Err.Error(), nil}")
				fmt.Fprintln(out, "\t\tdefault:")
				fmt.Fprintln(out, "\t\t\tw.WriteHeader(http.StatusInternalServerError)")
				fmt.Fprintln(out, "\t\t\tar = apiResponse{e.Error(), nil}")
				fmt.Fprintln(out, "\t\t}")
				fmt.Fprintln(out, "\t} else {")
				fmt.Fprintln(out, "\t\tar = apiResponse{\"\", srvResponse}")
				fmt.Fprintln(out, "\t}")

				fmt.Fprintln(out)
				fmt.Fprintln(out, "\tj, err := encodeJson(ar)")
				fmt.Fprintln(out, "\tif err != nil {")
				fmt.Fprintln(out, "\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)")
				fmt.Fprintln(out, "\t\treturn")
				fmt.Fprintln(out, "\t}")
				fmt.Fprintln(out, "\tw.Write(j)")
				fmt.Fprintln(out, "}")
			}
		} // end of handler funcs

		// generate the main mux 'ServerHTTP' functions
		fmt.Fprintf(out, "func (%s %s) ServeHTTP(w http.ResponseWriter, r *http.Request) {\n", muxPrefix, k)
		fmt.Fprintln(out, "\tswitch r.URL.Path {")
		for _, e := range v {
			fmt.Fprintf(out, "\tcase \"%s\":\n", e.Path)
			fmt.Fprintf(out, "\t\tf := %s.%s\n", muxPrefix, e.handlerFuncName())
			if e.Method != "" {
				fmt.Fprintf(out, "\t\tf = requestMethodMiddleWare(f, \"%s\")\n", e.Method)
			}
			if e.Auth { // optional part
				fmt.Fprintln(out, "\t\tf = authenticatedMiddleWare(f)")
			}
			fmt.Fprintln(out, "\t\tf(w, r)")
		}

		fmt.Fprintln(out, "\tdefault:\n") // default case
		fmt.Fprintln(out, "\t\tw.WriteHeader(http.StatusNotFound)")
		fmt.Fprintln(out, "\t\tar := apiResponse{\"unknown method\", nil}")
		fmt.Fprintln(out, "\t\tj, err := encodeJson(ar)")
		fmt.Fprintln(out, "\t\tif err != nil {")
		fmt.Fprintln(out, "\t\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)")
		fmt.Fprintln(out, "\t\t\treturn")
		fmt.Fprintln(out, "\t\t}")
		fmt.Fprintln(out, "\t\tw.Write(j)")
		fmt.Fprintln(out, "\t}")
		fmt.Fprintln(out, "}")

		// ---

		for _, e := range v {
			for _, p := range e.Params {
				fmt.Fprintf(out, "func (%s %s) Validate() error {\n", muxPrefix, p.Name)
				for _, f := range p.Fields {
					for _, v := range f.Validators {
						fmt.Fprintln(out, Indent(v.Render(f), "\t"))
					}
				}

				fmt.Fprintln(out, "\treturn nil")
				fmt.Fprintln(out, "}")
			}
		}
	}

	// ---

	// print headers
	fmt.Fprintln(out, "type apiResponse struct {")
	fmt.Fprintln(out, "\tError    string      `json:\"error\"`")
	fmt.Fprintln(out, "\tResponse interface{} `json:\"response,omitempty\"`")
	fmt.Fprintln(out, "}")
	fmt.Fprintln(out)

	fmt.Fprintln(out, jsonFacilties()) // json

	// add middlewares
	fmt.Fprintln(out, authenticatedMiddleware())
	fmt.Fprintln(out, requestMethodMiddleware())
} // end of main func

// ---

// middleware functions
func authenticatedMiddleware() string {
	return `func authenticatedMiddleWare(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authToken := r.Header.Get("X-Auth")
		if authToken != "100500" {
			// no authentication found
			ar := apiResponse{"unauthorized", nil} // blank error to be present inside
	
			j, err := encodeJson(ar)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError) // json serialization error
				return
			}
	
			w.WriteHeader(http.StatusForbidden) // 403
			w.Write(j)
	
			return // stop execution
		}
	
		// next
		h(w, r)
	})
}`
}

func requestMethodMiddleware() string {
	return `func requestMethodMiddleWare(h http.HandlerFunc, expectedReqMethod string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != expectedReqMethod {
			// no authentication found
			ar := apiResponse{"bad method", nil} // blank error to be present inside
	
			j, err := encodeJson(ar)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError) // json serialization error
				return
			}
	
			w.WriteHeader(http.StatusNotAcceptable) // 406
			w.Write(j)
	
			return // stop execution
		}
	
		// next
		h(w, r)
	})
}`
}

// end of middleware functions

func jsonFacilties() string {
	return `func encodeJson(content interface{}) ([]byte, error) {
	b := &bytes.Buffer{}
	c := json.NewEncoder(b) // new json encoder
	c.SetEscapeHTML(false)
	err := c.Encode(content) // -> json
	
	if err != nil {
		return nil, err
	}
	
	return b.Bytes(), nil
}
	
func throwBadRequest(w http.ResponseWriter, errorMessage string) {
	w.WriteHeader(http.StatusBadRequest) // 400 (bad request)
	
	ar := apiResponse{errorMessage, nil}
	j, err := encodeJson(ar)
	
	if err != nil { // json encoding error
		http.Error(w, err.Error(), http.StatusInternalServerError) // json serialization error
		return
	}
	
	w.Write(j) // flush
}`
}

// ---

// helper function to indent multiple (primarily) texts
func Indent(input string, indenter string) string {
	s := strings.Replace(input, "\n", "\n"+indenter, -1)
	return indenter + s
}
