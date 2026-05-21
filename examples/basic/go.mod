module github.com/dnahilman/goten/examples/basic

go 1.23

require (
	github.com/dnahilman/goten v0.0.0
	github.com/dnahilman/goten/adapters/gorm v0.0.0
	github.com/dnahilman/goten/plugins/username v0.0.0
	gorm.io/driver/postgres v1.5.9
	gorm.io/gorm v1.25.12
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.5.5 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/crypto v0.28.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/text v0.19.0 // indirect
)

replace (
	github.com/dnahilman/goten => ../..
	github.com/dnahilman/goten/adapters/gorm => ../../adapters/gorm
	github.com/dnahilman/goten/plugins/username => ../../plugins/username
)
