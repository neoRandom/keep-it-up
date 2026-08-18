package service

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func IsPasswordValid(password string) error {
	if strings.ContainsRune(password, ' ') {
		return fmt.Errorf("Password cannot have whitespaces")
	}

	if len(password) < 6 {
		return fmt.Errorf("Password cannot have less than 6 characters")
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
