package boot

import (
	"fmt"
	"os"

	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/pkg/tools/result"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// postgreSQLBoot returnsand execute DB related config and processes
func PostgreSQLBoot(dbentity ...any) result.R[gorm.DB] {
	schema := os.Getenv(consts.DB_SCHEMA)
	if schema == "" {
		schema = "public"
	}

	// Initialize the PostgreSQL connection using Gorm
	db, err := gorm.Open(postgres.Open(fmt.Sprintf("%s?search_path=%s", os.Getenv(consts.DB_PATH), schema)), &gorm.Config{})
	if err != nil {
		return result.Error[gorm.DB](err)
	}

	// Try creating the schema
	err = db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)).Error
	if err != nil {
		return result.Error[gorm.DB](err)
	}
	if err = db.AutoMigrate(dbentity...); err != nil {
		return result.Error[gorm.DB](err)
	}
	return result.Ok(db)
}
