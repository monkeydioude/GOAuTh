package boot

import (
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"

	"gorm.io/gorm"
)

// dropLegacyColumns removes columns AutoMigrate leaves behind.
func dropLegacyColumns(db *gorm.DB) error {
	m := db.Migrator()
	// replaced by the sessions table: one refresh token hash per login
	if !m.HasColumn(&entities.User{}, "refresh_token") {
		return nil
	}
	return m.DropColumn(&entities.User{}, "refresh_token")
}
