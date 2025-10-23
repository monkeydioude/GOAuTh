module github.com/monkeydioude/goauth

go 1.25.0

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/calqs/gopkg/env v1.0.2
	github.com/calqs/gopkg/gormslog v1.0.0
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/monkeydioude/goauth/pkg/tools v0.0.0-00010101000000-000000000000
	github.com/monkeydioude/heyo v0.0.0-20250126200040-f6155d390de8
	github.com/oklog/run v1.1.0
	github.com/stretchr/testify v1.10.0
	golang.org/x/crypto v0.38.0
	google.golang.org/grpc v1.74.2
	google.golang.org/protobuf v1.36.6
	gorm.io/driver/postgres v1.5.9
	gorm.io/gorm v1.30.3
)

require github.com/calqs/gopkg/dt v1.0.2 // indirect

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.5.5 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/monkeydioude/goauth/pkg/http v0.0.0-20250805064623-fd5a998f420e
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/monkeydioude/goauth/pkg/http => ./pkg/http

replace github.com/monkeydioude/goauth/pkg/tools => ./pkg/tools
