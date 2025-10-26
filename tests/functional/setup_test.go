package functional

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/monkeydioude/goauth/internal/api/handlers"
	"github.com/monkeydioude/goauth/internal/config/boot"
	"github.com/monkeydioude/goauth/internal/domain/entities"
	"github.com/monkeydioude/goauth/internal/domain/entities/constraints"
	v1 "github.com/monkeydioude/goauth/pkg/grpc/v1"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/gorm"
)

func setup() (*handlers.Layout, *gorm.DB, time.Time) {
	if os.Getenv("DB_PATH") == "" {
		fmt.Println("[INFO] no DB_PATH env found. Fallback on postgres://test:test@0.0.0.0:5445/test_db (make run-test-db)")
		os.Setenv("DB_PATH", "postgres://test:test@0.0.0.0:5445/test_db")
	}
	os.Setenv("JWT_SECRET", "test")
	// init layout
	res := boot.LayoutBoot([]any{entities.NewEmptyUser()}, []constraints.LoginConstraint{constraints.EmailConstraint}, []constraints.PasswordConstraint{})
	if res.IsErr() {
		log.Fatalf("Could not boot layout: %s", res.Error.Error())
	}
	layout := res.Result()
	layout.DB.Exec("TRUNCATE TABLE users")

	timeRef := time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	// change to the JWTFactory, so we can manipulate
	// its time reference logic
	layout.JWTFactory.TimeFn = func() time.Time {
		return timeRef
	}
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	return layout, layout.DB, timeRef
}

func setupRPC(t *testing.T, layout *handlers.Layout) *grpc.ClientConn {
	server := grpc.NewServer()
	v1.RegisterJWTServer(server, v1.NewJWTRPCHandler(layout))
	v1.RegisterAuthServer(server, v1.NewAuthRPCHandler(layout))
	v1.RegisterUserServer(server, v1.NewUserRPCHandler(layout))

	lis := bufconn.Listen(1024 * 1024)
	go func() {
		if err := server.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(
			func(ctx context.Context, _ string) (net.Conn, error) {
				return lis.Dial()
			}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.NoError(t, err)
	return conn
}

func cleanup(layout *handlers.Layout) {
	if layout.DB != nil {
		if sql, err := layout.DB.DB(); err != nil {
			sql.Close()
		}
	}
}
