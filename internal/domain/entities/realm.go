package entities

type Realm struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"-"`
	Name        string `gorm:"unique;not null" json:"name"`
	Description string `gorm:"not null;default:''" json:"description,omitempty"`
}
