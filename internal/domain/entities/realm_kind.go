package entities

const (
	// RealmKindHuman holds people, who sign up and log in with a password. It is
	// the kind of every realm from before kinds existed.
	RealmKindHuman string = "human"
	// RealmKindService holds accounts that are not people, made by a trusted
	// backend. They have no password, so no password flow applies to them.
	RealmKindService string = "service"
)
