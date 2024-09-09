package boot

import (
	"GOAuTh/pkg/tools/result"
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func postgreSQLBoot(dbEntities ...any) result.R[gorm.DB] {
	schema := os.Getenv("DB_SCHEMA")
	if schema == "" {
		schema = "public"
	}
	// Initialize the PostgreSQL connection using Gorm
	db, err := gorm.Open(postgres.Open(fmt.Sprintf("%s?search_path=%s", os.Getenv("DB_PATH"), schema)), &gorm.Config{})
	if err != nil {
		return result.Error[gorm.DB](err)
	}

	// Try creating the schema
	err = db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", os.Getenv("DB_SCHEMA"))).Error
	if err != nil {
		return result.Error[gorm.DB](err)
	}
	if err = db.AutoMigrate(dbEntities...); err != nil {
		return result.Error[gorm.DB](err)
	}
	return result.Ok(db)
}
