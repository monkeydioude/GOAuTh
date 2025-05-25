package main

//go:generate go run ./constraint_maker.go ./output.go ./errors.go ./generator.go

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func getFiles(dirPath string) []string {
	files := []string{}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files
}

func getValidatorTag(field *ast.Field) (string, bool) {
	if field == nil || field.Tag == nil {
		return "", false
	}
	tagLit := field.Tag.Value
	tagLit = strings.Trim(tagLit, "`")
	st := reflect.StructTag(tagLit)
	return st.Lookup("validator")
}

func processEntity(
	structType *ast.StructType,
	typeSpec *ast.TypeSpec,
	constMakerBase constraintsSliceMaker,
) {
	constMaker := constMakerBase.WithStruct(typeSpec.Name.Name)
	for _, field := range structType.Fields.List {
		tag, ok := getValidatorTag(field)
		if !ok {
			continue
		}
		if tag == "" && len(field.Names) > 0 {
			tag = field.Names[0].Name
		}
		constMaker = constMakerBase.WithField(tag)
		fieldTypeStr := typeStringFromExpr(field.Type)
		writeConstraintMaker(constMaker, fieldTypeStr)
	}
}

func main() {
	writeComments()
	writePackage()
	writeStructBegin()
	dirPath := os.Getenv("INPUT_DIR")
	for _, fileName := range getFiles(dirPath) {
		inputFile := filepath.Join(dirPath, fileName)
		fset := token.NewFileSet()

		fileAst, err := parser.ParseFile(fset, inputFile, nil, parser.AllErrors)
		if err != nil {
			panic(err)
		}
		if fileAst.Name == nil {
			panic(fmt.Errorf("fileAst.Name: %w", ErrNilPointer))
		}
		constMaker := constraintsSliceMaker{
			Pkg: cases.Title(language.English).String(fileAst.Name.Name),
		}

		for _, decl := range fileAst.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				processEntity(structType, typeSpec, constMaker)
			}
		}
	}
	writeStructEnd()
	writeAppendFuncs()
	formatted, err := format.Source([]byte(sb.String()))
	if err != nil {
		panic(err)
	}
	fmt.Println(string(formatted))
	err = os.WriteFile(outputFile, formatted, 0644)
	if err != nil {
		panic(err)
	}
	// 	genDecl, ok := decl.(*ast.GenDecl)
	// 	if !ok {
	// 		continue
	// 	}
	// 	for _, spec := range genDecl.Specs {
	// 		typeSpec, ok := spec.(*ast.TypeSpec)
	// 		if !ok {
	// 			continue
	// 		}

	// 		// If it's called "User", let's capture its fields.
	// 		if typeSpec.Name.Name == "User" {
	// 			structType, ok := typeSpec.Type.(*ast.StructType)
	// 			if !ok {
	// 				continue
	// 			}
	// 			userFields = structType.Fields.List
	// 		}
	// 	}
	// }

	// // 3) Build the generated code in a string builder
	// var sb strings.Builder

	// // Write the package line
	// sb.WriteString(fmt.Sprintf("package %s\n\n", pkgName))

	// // We don't need any imports for this simple example.
	// // If you DO, you'd do:
	// // sb.WriteString("import (\n\t\"fmt\"\n)\n\n")

	// // 4) For each field in `User`, generate a type definition
	// //    like: type ModelsUserLoginConstraints []func(string, string)
	// for _, field := range userFields {
	// 	// Usually each field has exactly one name.
	// 	if len(field.Names) == 0 {
	// 		continue
	// 	}
	// 	fieldName := field.Names[0].Name

	// 	// Figure out the *literal type name* (e.g. "string", "int", etc.)
	// 	fieldTypeStr := typeStringFromExpr(field.Type)

	// 	// Title-case the package name to build something like "ModelsUserLoginConstraints"
	// 	// (Though many prefer capitalizing only the struct and field, etc. — up to you!)
	// 	typeDefName := fmt.Sprintf("%s%s%sConstraints",
	// 		strings.Title(pkgName), // e.g. "Models"
	// 		"User",                 // struct name
	// 		fieldName,              // e.g. "Login"
	// 	)

	// 	// Write out:  type ModelsUserLoginConstraints []func(<T>, <T>)
	// 	sb.WriteString(fmt.Sprintf("type %s []func(%s, %s)\n\n", typeDefName, fieldTypeStr, fieldTypeStr))
	// }

	// // 5) Format the generated code for nice indentation
	// formatted, err := format.Source([]byte(sb.String()))
	// if err != nil {
	// 	panic(err)
	// }

	// // 6) Write the result to a new .go file
	// err = os.WriteFile(outputFile, formatted, 0644)
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println("Code generated successfully at", outputFile)
}

// Helper: turn an AST expr (e.g. an *ast.Ident) into a string like "string", "int"...
func typeStringFromExpr(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name // e.g. "string", "int"
	case *ast.SelectorExpr:
		// e.g. if it's something like "time.Time"
		// then X is an *ast.Ident with Name="time", and Sel is "Time"
		xIdent, ok := t.X.(*ast.Ident)
		if ok {
			return fmt.Sprintf("%s.%s", xIdent.Name, t.Sel.Name)
		}
	case *ast.StarExpr:
		// pointer to something
		return "*" + typeStringFromExpr(t.X)
	case *ast.ArrayType:
		return "[]" + typeStringFromExpr(t.Elt)
	}
	// fallback
	return "any"
}
