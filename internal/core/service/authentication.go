package service

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func IsPasswordValid(password string) error {
	for _, r := range password {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return fmt.Errorf("password may only contain letters and digits")
		}
	}

	if len(password) < 6 {
		return fmt.Errorf("password cannot have less than 6 characters")
	}

	return nil
}

func GeneratePasswordHash(password string) (string, error) {
	if err := IsPasswordValid(password); err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}
