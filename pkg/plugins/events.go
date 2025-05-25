package plugins

type Event string

const (
	OnUserCreation         Event = "event.on_user_creation"
	BootConstraintEmail    Event = "boot.constraint.email"
	BootConstraintPassword Event = "boot.constraint.password"
)

type Step string

const (
	Before Step = "before"
	After  Step = "after"
)
