// this file is auto-generated. Do not edit
package constraints

type Constraints struct {
	EntitiesLoginConstraints    []func(string, string)
	EntitiesPasswordConstraints []func(string, string)
}

func NewConstraints() *Constraints {
	return &Constraints{}
}

func (c *Constraints) AddEntitiesLoginConstraint(constraint func(string, string)) {
	c.EntitiesLoginConstraints = append(c.EntitiesLoginConstraints, constraint)
}

func (c *Constraints) AddEntitiesPasswordConstraint(constraint func(string, string)) {
	c.EntitiesPasswordConstraints = append(c.EntitiesPasswordConstraints, constraint)
}
