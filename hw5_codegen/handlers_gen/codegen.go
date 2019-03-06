package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
)

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

		// g, ok := node.(*ast.GenDecl)
		// if !ok {
		// 	// fmt.Printf("SKIP %v is not *ast.GenDecl\n", f)
		// 	continue
		// }

		// // ---

		// for _, spec := range g.Specs {
		// 	_, ok := spec.(*ast.TypeSpec)
		// 	if !ok {
		// 		// fmt.Printf("SKIP %T is not ast.TypeSpec\n", spec)
		// 		continue
		// 	} else {
		// 		// fmt.Println(currType)
		// 	}
		// }

		f, ok := node.(*ast.FuncDecl)
		if !ok {
			// fmt.Printf("SKIP %v is not *ast.GenDecl\n", f)
			continue
		} else {
			// fmt.Println(f)
			if f.Doc == nil {
				continue
			}

			fmt.Println(f.Name)

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

				fmt.Println(comment.Text)
			}
		}

		fmt.Println("---")
	}
}
