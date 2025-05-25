package main

import (
	"fmt"
	"os"
	"strings"
)

var sb strings.Builder
var imports strings.Builder
var outputFile = os.Getenv("OUTPUT_FILE")
var outputPkg = os.Getenv("OUTPUT_PKG")
var appendFuncs = []string{}

const structName string = "Constraints"

const appendFuncTemplate = `func (c *%s) %s (constraint func(%s, %s)) {
	c.%s = append(c.%s, constraint)
}`

const structEndTemplate = `}
func NewConstraints() *Constraints {
	return &Constraints{}
}`

const structBeginTemplate = `type %s struct {
`

const structMakerFieldTemplate = `	%s []func(%s, %s)
`

func init() {
	if outputFile == "" {
		outputFile = "./constraints.go"
	}
	if outputPkg == "" {
		outputPkg = "constraints"
	}
}
func writeComments() {
	sb.WriteString("// this file is auto-generated. Do not edit\n")
}

func writePackage() {
	sb.WriteString(fmt.Sprintf("package %s\n\n", outputPkg))
}

func writeConstraintMaker(constMaker constraintsSliceMaker, fieldType string) {
	typeName := fmt.Sprintf("%sConstraints", constMaker.String())
	sb.WriteString(fmt.Sprintf(structMakerFieldTemplate, typeName, fieldType, fieldType))
	funcName := fmt.Sprintf("Add%sConstraint", constMaker.String())
	appendFuncs = append(appendFuncs, fmt.Sprintf(appendFuncTemplate, structName, funcName, fieldType, fieldType, typeName, typeName))
}

func writeAppendFuncs() {
	for _, fnStr := range appendFuncs {
		sb.WriteString(fnStr)
		sb.WriteString("\n\n")
	}
}

func writeStructBegin() {
	sb.WriteString(fmt.Sprintf(structBeginTemplate, structName))
}

func writeStructEnd() {
	sb.WriteString(structEndTemplate)
	sb.WriteString("\n\n")
}
