module github.com/dnahilman/goten/cmd/goten

go 1.25.0

require (
	github.com/dnahilman/goten v0.2.0
	github.com/dnahilman/goten/plugins/admin v0.0.0
	github.com/dnahilman/goten/plugins/oauth v0.2.0
	github.com/dnahilman/goten/plugins/username v0.2.0
	github.com/urfave/cli/v3 v3.9.0
	gopkg.in/yaml.v3 v3.0.1
)

// admin has no published tag yet; resolve it locally during development.
// REMOVE this replace and bump the require above to the real version at release.
replace github.com/dnahilman/goten/plugins/admin => ../../plugins/admin

require (
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
)
