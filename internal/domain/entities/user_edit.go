package entities

import "github.com/monkeydioude/goauth/internal/domain/models"

type EditUserPayload struct {
	Password    string              `json:"password"`
	NewLogin    *string             `json:"new_login"`
	NewPassword *string             `json:"new_password"`
	UserParams  *models.UsersParams `json:"-"`
}
