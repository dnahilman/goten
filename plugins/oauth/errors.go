package oauth

import "errors"

var (
	ErrProviderNotFound     = errors.New("oauth: provider not found")
	ErrStateInvalid         = errors.New("oauth: invalid or expired state")
	ErrMissingCode          = errors.New("oauth: authorization code missing")
	ErrAccountNotLinked     = errors.New("oauth: account exists but is not linked to this provider")
	ErrSignUpDisabled       = errors.New("oauth: sign up is disabled")
	ErrAccountAlreadyLinked = errors.New("oauth: provider account already linked to another user")
	ErrEmailMismatch        = errors.New("oauth: provider email does not match the current user")
	ErrCannotUnlinkLast     = errors.New("oauth: cannot unlink the only login method")
	ErrEmailNotFound        = errors.New("oauth: provider did not return an email")
	ErrAccountNotFound      = errors.New("oauth: account not found")
	ErrNoRefreshToken       = errors.New("oauth: no refresh token stored for this account")
	ErrRefreshNotSupported  = errors.New("oauth: provider does not support token refresh")
	ErrIDTokenNotSupported  = errors.New("oauth: provider does not support id_token sign-in")
	ErrIDTokenInvalid       = errors.New("oauth: invalid id_token")
	ErrUntrustedRedirect    = errors.New("oauth: callbackURL is not a trusted origin")
)
