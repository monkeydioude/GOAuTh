package logs

import (
	"context"
	"log/slog"
	"os"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"gorm.io/gorm"
)

type xRequestIDKey struct{}

// WithXRequestID returns a context carrying the request's X-Request-ID,
// so every *Context log written with it carries the id.
func WithXRequestID(ctx context.Context, xRequestID string) context.Context {
	return context.WithValue(ctx, xRequestIDKey{}, xRequestID)
}

// XRequestID returns the X-Request-ID put in the context by the middlewares,
// "" outside of a request (boot, jobs).
func XRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	xRequestID, _ := ctx.Value(xRequestIDKey{}).(string)
	return xRequestID
}

// DBContext returns the context a *gorm.DB was given through WithContext,
// so services handed a db can log with the request's X-Request-ID.
func DBContext(db *gorm.DB) context.Context {
	if db == nil || db.Statement == nil || db.Statement.Context == nil {
		return context.Background()
	}
	return db.Statement.Context
}

// NewHandler wraps h so it adds the context's X-Request-ID to every record.
func NewHandler(h slog.Handler) slog.Handler {
	return contextHandler{Handler: h}
}

// contextHandler adds the context's X-Request-ID to every record logged
// through a *Context method.
type contextHandler struct {
	slog.Handler
}

var _ slog.Handler = (*contextHandler)(nil)

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if xRequestID := XRequestID(ctx); xRequestID != "" {
		r.AddAttrs(slog.String(consts.X_REQUEST_ID_LABEL, xRequestID))
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{Handler: h.Handler.WithGroup(name)}
}

// SetupSlogger makes the default slog logger write JSON, readable text on DEV,
// with the X-Request-ID of the context. The standard log package goes through it too.
// Call it before PostgreSQLBoot: gorm keeps the default logger it is given.
func SetupSlogger() {
	if os.Getenv("ENV") == "DEV" {
		slog.SetDefault(slog.New(NewHandler(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)))
		return
	}
	slog.SetDefault(slog.New(NewHandler(
		slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}),
	)))
}
