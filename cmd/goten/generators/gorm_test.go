package generators

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	goten "github.com/dnahilman/goten"
)

// schemaWith builds a merged schema like the CLI's mergeSchema, for the given plugins.
func schemaWith(plugins ...string) map[string]goten.TableSchema {
	s := goten.CoreSchema()
	for _, p := range plugins {
		switch p {
		case "username":
			u := s["users"]
			u.Fields = append(u.Fields, goten.FieldDef{Name: "username", Type: "text", Unique: true})
			s["users"] = u
		case "oauth":
			a := s["accounts"]
			a.Fields = append(a.Fields,
				goten.FieldDef{Name: "access_token", Type: "text"},
				goten.FieldDef{Name: "refresh_token", Type: "text"},
				goten.FieldDef{Name: "id_token", Type: "text"},
				goten.FieldDef{Name: "access_token_expires_at", Type: "timestamp"},
				goten.FieldDef{Name: "refresh_token_expires_at", Type: "timestamp"},
				goten.FieldDef{Name: "scope", Type: "text"},
			)
			s["accounts"] = a
		}
	}
	return s
}

func generate(t *testing.T, schema map[string]goten.TableSchema) string {
	t.Helper()
	g, ok := Get("gorm")
	if !ok {
		t.Fatal("gorm generator not registered")
	}
	res, err := g.Generate(schema, Options{Package: "authmodels", TableOrder: goten.CoreTableOrder})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// Output must be valid, parseable Go.
	if _, err := parser.ParseFile(token.NewFileSet(), "auth_models.go", res.Code, parser.AllErrors); err != nil {
		t.Fatalf("generated code does not parse: %v\n%s", err, res.Code)
	}
	return res.Code
}

func TestGorm_CoreOnly(t *testing.T) {
	code := generate(t, schemaWith())
	for _, want := range []string{
		"package authmodels",
		"type User struct",
		"type Session struct",
		"type Account struct",
		"type Verification struct",
		`func (Verification) TableName() string { return "verification" }`,
		`func (User) TableName() string { return "users" }`,
		"func AllModels() []any",
		`gorm:"column:id;primaryKey"`,
		`gorm:"column:email;uniqueIndex;not null"`,
	} {
		if !strings.Contains(code, want) {
			t.Errorf("core output missing %q", want)
		}
	}
	// No oauth/username columns when those plugins are absent.
	if strings.Contains(code, "access_token") {
		t.Errorf("access_token must be absent without the oauth plugin")
	}
	if strings.Contains(code, "Username") {
		t.Errorf("Username must be absent without the username plugin")
	}
}

func TestGorm_WithPlugins(t *testing.T) {
	code := generate(t, schemaWith("username", "oauth"))
	// gofmt aligns struct columns, so collapse whitespace for field+type checks.
	norm := strings.Join(strings.Fields(code), " ")
	for _, want := range []string{"Username *string", "AccessToken *string"} {
		if !strings.Contains(norm, want) {
			t.Errorf("plugin output missing %q (optional fields must be pointers)", want)
		}
	}
	for _, want := range []string{`gorm:"column:username;uniqueIndex"`, `gorm:"column:access_token"`} {
		if !strings.Contains(code, want) {
			t.Errorf("plugin output missing tag %q", want)
		}
	}
}

func TestGorm_CompositeUniqueAndFK(t *testing.T) {
	code := generate(t, schemaWith("oauth"))
	// Composite unique: provider_id + account_id share one named index.
	if strings.Count(code, "uniqueIndex:uq_accounts_provider_id_account_id") != 2 {
		t.Errorf("composite unique index must appear on both provider_id and account_id\n%s", code)
	}
	// Foreign key association on user_id → User.
	if !strings.Contains(code, "foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE") {
		t.Errorf("missing FK association for user_id")
	}
}

func TestGorm_PascalCaseInitialisms(t *testing.T) {
	code := generate(t, schemaWith("oauth"))
	for _, want := range []string{"UserID ", "IDToken ", "AccessTokenExpiresAt "} {
		if !strings.Contains(code, want) {
			t.Errorf("expected idiomatic field name %q", want)
		}
	}
}
