package boot

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/calqs/gopkg/env"
	"github.com/calqs/gopkg/gormslog"
	"github.com/joho/godotenv"
	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/pkg/tools/result"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DbEnv struct {
	LogLevel             string `env:"DB_LOG_LEVEL,?error"`
	SlowQueryThresholdMs int    `env:"DB_SLOW_QUERY_LOG_MS,?20"`
}

func getLogLevel(ev string) logger.LogLevel {
	switch ev {
	case "info":
		return logger.Info
	case "warn":
		return logger.Warn
	case "error":
		return logger.Error
	default:
		return logger.Error
	}
}

// postgreSQLBoot returns and execute DB related config and processes
func PostgreSQLBoot(dbentity ...any) result.R[gorm.DB] {
	godotenv.Load()
	config, err := env.ParseEnv[DbEnv]()
	if err != nil {
		return result.Error[gorm.DB](err)
	}
	schema := os.Getenv(consts.DB_SCHEMA)
	if schema == "" {
		schema = "public"
	}
	gormLogLevel := getLogLevel(config.LogLevel)
	slog.Info(
		"connection to postgres server",
		"schema", schema,
		"logLevel", config.LogLevel,
		"gormLogLevel", gormLogLevel,
		"slow_query_threshold_ms", config.SlowQueryThresholdMs,
	)
	// Initialize the PostgreSQL connection using Gorm
	db, err := gorm.Open(postgres.Open(fmt.Sprintf("%s?search_path=%s", os.Getenv(consts.DB_PATH), schema)), &gorm.Config{
		Logger: gormslog.New(slog.Default(), gormLogLevel, 20*time.Millisecond),
	})
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
