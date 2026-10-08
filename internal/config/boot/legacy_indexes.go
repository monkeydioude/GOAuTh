package boot

import (
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"

	"gorm.io/gorm"
)

// dropLegacyIndexes removes indexes AutoMigrate leaves behind.
func dropLegacyIndexes(db *gorm.DB) error {
	m := db.Migrator()
	// replaced by idx_realm_login_active: a login is unique within its realm, not across realms
	if !m.HasIndex(&entities.User{}, "idx_login_active") {
		return nil
	}
	return m.DropIndex(&entities.User{}, "idx_login_active")
}
