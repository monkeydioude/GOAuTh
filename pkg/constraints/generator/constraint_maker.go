package main

import "fmt"

type constraintsSliceMaker struct {
	Pkg    string
	Struct string
	Field  string
}

func (c constraintsSliceMaker) String() string {
	return fmt.Sprintf("%s%s%s", c.Pkg, c.Struct, c.Field)
}

func (c constraintsSliceMaker) WithStruct(strct string) constraintsSliceMaker {
	return constraintsSliceMaker{
		Pkg:    c.Pkg,
		Struct: strct,
		Field:  c.Field,
	}
}

func (c constraintsSliceMaker) WithField(field string) constraintsSliceMaker {
	return constraintsSliceMaker{
		Pkg:    c.Pkg,
		Struct: c.Struct,
		Field:  field,
	}
}
