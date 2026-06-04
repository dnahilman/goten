package goten

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

// validate is the shared go-playground validator instance used for request and
// config validation across the core package.
var validate = validator.New(validator.WithRequiredStructEnabled())

// passwordCode maps a password-length validation error to the existing HTTP
// error code: a failing "max" rule → PASSWORD_TOO_LONG, otherwise PASSWORD_TOO_SHORT.
func passwordCode(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, fe := range ve {
			if fe.Tag() == "max" {
				return "PASSWORD_TOO_LONG"
			}
		}
	}
	return "PASSWORD_TOO_SHORT"
}
