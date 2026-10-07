// job-session-purge deletes for good the sessions revoked or expired more than
// SESSION_RETENTION_DAYS ago, then exits. Schedule it daily.
package main

import (
	"log"
	"log/slog"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/config/boot"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
)

func main() {
	sessionEnv, err := boot.SessionBoot()
	if err != nil {
		log.Fatal(err)
	}
	// no entities: the API binary owns the migrations
	dbRes := boot.PostgreSQLBoot()
	if dbRes.IsErr() {
		log.Fatal(dbRes.Error)
	}
	cutoff := time.Now().Add(-sessionEnv.Retention())
	purged, err := services.PurgeSessions(dbRes.Result(), cutoff)
	if err != nil {
		log.Fatal(err)
	}
	slog.Info("purged old sessions", "count", purged, "cutoff", cutoff)
}
