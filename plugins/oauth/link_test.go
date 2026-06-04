package oauth

import (
	"context"
	"errors"
	"testing"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/models"
)

func TestHandleOAuthUserInfo_NewUserAutoSignup(t *testing.T) {
	p, a := newTestPlugin(t, Options{})
	info := &UserInfo{ID: "g1", Email: "new@example.com", EmailVerified: true, Name: "New"}

	res, err := p.handleOAuthUserInfo(context.Background(), "google", info, &Tokens{AccessToken: "AT"}, false)
	if err != nil {
		t.Fatalf("handleOAuthUserInfo: %v", err)
	}
	if !res.IsRegister {
		t.Errorf("expected IsRegister=true for new user")
	}
	if res.User.Email != "new@example.com" {
		t.Errorf("email = %q", res.User.Email)
	}
	acc, _ := a.InternalAdapter().FindAccountByProviderAndID(context.Background(), "google", "g1")
	if acc == nil {
		t.Errorf("account should have been created")
	}
}

func TestHandleOAuthUserInfo_ReturningUser(t *testing.T) {
	p, a := newTestPlugin(t, Options{})
	ctx := context.Background()
	user := mustUser(t, a, "ret@example.com", true)
	if _, err := a.InternalAdapter().CreateAccount(ctx, user.ID, "g2", "google", nil); err != nil {
		t.Fatal(err)
	}

	res, err := p.handleOAuthUserInfo(ctx, "google", &UserInfo{ID: "g2", Email: "ret@example.com"}, &Tokens{}, false)
	if err != nil {
		t.Fatalf("handleOAuthUserInfo: %v", err)
	}
	if res.IsRegister {
		t.Errorf("expected IsRegister=false for returning user")
	}
	if res.User.ID != user.ID {
		t.Errorf("user id = %q want %q", res.User.ID, user.ID)
	}
}

func TestHandleOAuthUserInfo_LinkByVerifiedEmail(t *testing.T) {
	p, a := newTestPlugin(t, Options{})
	ctx := context.Background()
	user := mustUser(t, a, "link@example.com", true) // local verified

	res, err := p.handleOAuthUserInfo(ctx, "google", &UserInfo{ID: "g3", Email: "link@example.com", EmailVerified: true}, &Tokens{}, false)
	if err != nil {
		t.Fatalf("handleOAuthUserInfo: %v", err)
	}
	if res.User.ID != user.ID {
		t.Errorf("should link to existing user")
	}
	acc, _ := a.InternalAdapter().FindAccountByProviderAndID(ctx, "google", "g3")
	if acc == nil || acc.UserID != user.ID {
		t.Errorf("account should link to existing user")
	}
}

func TestHandleOAuthUserInfo_RejectUnverifiedLocal(t *testing.T) {
	p, a := newTestPlugin(t, Options{})
	ctx := context.Background()
	mustUser(t, a, "unverified-local@example.com", false) // local NOT verified

	_, err := p.handleOAuthUserInfo(ctx, "google", &UserInfo{ID: "g4", Email: "unverified-local@example.com", EmailVerified: true}, &Tokens{}, false)
	if !errors.Is(err, ErrAccountNotLinked) {
		t.Errorf("err = %v, want ErrAccountNotLinked", err)
	}
}

func TestHandleOAuthUserInfo_RejectUnverifiedProvider(t *testing.T) {
	p, a := newTestPlugin(t, Options{})
	ctx := context.Background()
	mustUser(t, a, "verified-local@example.com", true)

	// provider email NOT verified + provider not trusted → reject.
	_, err := p.handleOAuthUserInfo(ctx, "google", &UserInfo{ID: "g5", Email: "verified-local@example.com", EmailVerified: false}, &Tokens{}, false)
	if !errors.Is(err, ErrAccountNotLinked) {
		t.Errorf("err = %v, want ErrAccountNotLinked", err)
	}
}

func TestHandleOAuthUserInfo_TrustedProviderLinks(t *testing.T) {
	p, a := newTestPlugin(t, Options{TrustedProviders: []string{"google"}})
	ctx := context.Background()
	user := mustUser(t, a, "trusted@example.com", true) // local verified

	// provider email unverified but provider trusted → allowed (local verified).
	res, err := p.handleOAuthUserInfo(ctx, "google", &UserInfo{ID: "g6", Email: "trusted@example.com", EmailVerified: false}, &Tokens{}, false)
	if err != nil {
		t.Fatalf("trusted provider should link: %v", err)
	}
	if res.User.ID != user.ID {
		t.Errorf("should link to existing user")
	}
}

func TestHandleOAuthUserInfo_DisableSignUp(t *testing.T) {
	p, _ := newTestPlugin(t, Options{})
	_, err := p.handleOAuthUserInfo(context.Background(), "google", &UserInfo{ID: "g7", Email: "nope@example.com", EmailVerified: true}, &Tokens{}, true)
	if !errors.Is(err, ErrSignUpDisabled) {
		t.Errorf("err = %v, want ErrSignUpDisabled", err)
	}
}

func TestLinkToCurrentUser_EmailMismatch(t *testing.T) {
	p, a := newTestPlugin(t, Options{})
	ctx := context.Background()
	user := mustUser(t, a, "owner@example.com", true)

	err := p.linkToCurrentUser(ctx, user, "google", &UserInfo{ID: "g8", Email: "other@example.com"}, &Tokens{})
	if !errors.Is(err, ErrEmailMismatch) {
		t.Errorf("err = %v, want ErrEmailMismatch", err)
	}
}

func TestLinkToCurrentUser_AlreadyLinkedOther(t *testing.T) {
	p, a := newTestPlugin(t, Options{})
	ctx := context.Background()
	u1 := mustUser(t, a, "u1@example.com", true)
	u2 := mustUser(t, a, "u2@example.com", true)
	if _, err := a.InternalAdapter().CreateAccount(ctx, u1.ID, "shared", "google", nil); err != nil {
		t.Fatal(err)
	}

	err := p.linkToCurrentUser(ctx, u2, "google", &UserInfo{ID: "shared", Email: "u2@example.com"}, &Tokens{})
	if !errors.Is(err, ErrAccountAlreadyLinked) {
		t.Errorf("err = %v, want ErrAccountAlreadyLinked", err)
	}
}

func mustUser(t *testing.T, a *goten.Auth, email string, verified bool) *models.User {
	t.Helper()
	u, err := a.InternalAdapter().CreateUserWithExtra(context.Background(), email, "Test", verified, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}
