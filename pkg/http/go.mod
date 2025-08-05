module github.com/monkeydioude/goauth/pkg/http

go 1.24.0

require (
	github.com/monkeydioude/goauth/pkg/tools v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.74.2
)

require golang.org/x/sys v0.33.0 // indirect

replace github.com/monkeydioude/goauth/pkg/tools => ../tools
