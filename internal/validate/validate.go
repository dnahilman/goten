package validate

import (
	"fmt"
	"regexp"
)

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func IsValidEmail(email string) bool {
	return emailRe.MatchString(email)
}

func Password(password string, min, max int) error {
	l := len(password)
	if l < min {
		return fmt.Errorf("password must be at least %d characters", min)
	}
	if l > max {
		return fmt.Errorf("password must be at most %d characters", max)
	}
	return nil
}
