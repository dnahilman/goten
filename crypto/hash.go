package crypto

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const defaultBcryptCost = 12

func HashPassword(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), defaultBcryptCost)
	return string(hash), err
}

func VerifyPassword(hash, pwd string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	return err == nil, err
}
