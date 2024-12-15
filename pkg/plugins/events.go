package plugins

type Event string

const (
	OnUserCreation Event = "event.on_user_creation"
)

type Step string

const (
	Before Step = "before"
	After  Step = "after"
)
