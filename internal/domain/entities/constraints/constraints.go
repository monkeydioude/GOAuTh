package constraints

type EntityField func(string, *string) error

type LoginConstraint EntityField
type PasswordConstraint EntityField
