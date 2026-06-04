package oauth

import (
	"context"
	"strings"
	"time"

	goten "github.com/dnahilman/goten"
	"github.com/dnahilman/goten/models"
)

type oauthResult struct {
	User       *models.User
	IsRegister bool
}

// handleOAuthUserInfo creates or links the user for a completed OAuth sign-in,
// applying better-auth's account-linking safety: an existing user is only
// auto-linked to an unlinked provider account when the provider email is
// verified (or the provider is trusted) AND, by default, the local email is
// verified — guarding against account takeover.
func (p *Plugin) handleOAuthUserInfo(ctx context.Context, providerID string, info *UserInfo, tokens *Tokens, disableSignUp bool) (*oauthResult, error) {
	ia := p.auth.InternalAdapter()
	email := strings.ToLower(info.Email)

	existingAccount, err := ia.FindAccountByProviderAndID(ctx, providerID, info.ID)
	if err != nil {
		return nil, err
	}

	// Already linked → refresh tokens and return the owning user.
	if existingAccount != nil {
		if _, err := p.updateAccountTokens(ctx, existingAccount.ID, tokens); err != nil {
			return nil, err
		}
		user, err := ia.FindUserByID(ctx, existingAccount.UserID)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, ErrAccountNotFound
		}
		return &oauthResult{User: user}, nil
	}

	user, err := ia.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if user != nil {
		// User exists but the provider account is not linked → safety gate.
		trusted := p.opts.isTrustedProvider(providerID)
		if (!trusted && !info.EmailVerified) ||
			(p.opts.requireLocalEmailVerified() && !user.EmailVerified) ||
			!p.opts.linkingEnabled() {
			return nil, ErrAccountNotLinked
		}
		if err := p.createLinkedAccount(ctx, user.ID, providerID, info, tokens); err != nil {
			return nil, err
		}
		// Promote the local email to verified when the provider asserts it.
		if info.EmailVerified && !user.EmailVerified && email == strings.ToLower(user.Email) {
			if updated, err := ia.UpdateUser(ctx, user.ID, map[string]any{"email_verified": true}); err == nil && updated != nil {
				user = updated
			}
		}
		return &oauthResult{User: user}, nil
	}

	// No user → implicit sign-up unless disabled.
	if disableSignUp {
		return nil, ErrSignUpDisabled
	}
	extra := map[string]any{}
	if info.Image != "" {
		extra["image"] = info.Image
	}
	var newUser *models.User
	if err := ia.WithTransaction(ctx, func(txCtx context.Context) error {
		u, err := ia.CreateUserWithExtra(txCtx, email, info.Name, info.EmailVerified, extra)
		if err != nil {
			return err
		}
		if err := p.createLinkedAccount(txCtx, u.ID, providerID, info, tokens); err != nil {
			return err
		}
		newUser = u
		return nil
	}); err != nil {
		return nil, err
	}
	return &oauthResult{User: newUser, IsRegister: true}, nil
}

// linkToCurrentUser links a provider account to an already-authenticated user
// (the /link-social flow).
func (p *Plugin) linkToCurrentUser(ctx context.Context, user *models.User, providerID string, info *UserInfo, tokens *Tokens) error {
	ia := p.auth.InternalAdapter()
	existing, err := ia.FindAccountByProviderAndID(ctx, providerID, info.ID)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.UserID != user.ID {
			return ErrAccountAlreadyLinked
		}
		_, err := p.updateAccountTokens(ctx, existing.ID, tokens)
		return err
	}
	if !p.opts.AccountLinking.AllowDifferentEmails && strings.ToLower(info.Email) != strings.ToLower(user.Email) {
		return ErrEmailMismatch
	}
	return p.createLinkedAccount(ctx, user.ID, providerID, info, tokens)
}

func (p *Plugin) createLinkedAccount(ctx context.Context, userID, providerID string, info *UserInfo, tokens *Tokens) error {
	_, err := p.auth.InternalAdapter().CreateAccount(ctx, userID, info.ID, providerID, p.accountTokenFields(tokens))
	return err
}

func (p *Plugin) updateAccountTokens(ctx context.Context, accountID string, tokens *Tokens) (map[string]any, error) {
	fields := p.accountTokenFields(tokens)
	fields["updated_at"] = time.Now().UTC()
	return p.auth.Adapter().Update(ctx, "accounts", goten.Query{Where: []goten.Where{goten.EQ("id", accountID)}}, fields)
}
