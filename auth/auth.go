package auth

import (
	"encoding/hex"
	"errors"
	"strings"
	"unicode"

	"github.com/0sm1les/gopherbb/models"

	"golang.org/x/crypto/argon2"
)

var salt []byte

func ValidateUser(username string) (models.Username, error) {
	username = strings.TrimSpace(username)
	username = strings.ToLower(username)

	if len(username) > 16 {
		return "", errors.New("username over 16 characters")
	} else if len(username) < 3 {
		return "", errors.New("username under 3 characters")
	}

	for _, c := range username {
		if unicode.IsSpace(c) {
			return "", errors.New("username contains white space")
		}
		if !unicode.IsDigit(c) && !unicode.IsLetter(c) {
			return "", errors.New("username contains non-alphanumeric characters")
		}
	}
	return models.Username(username), nil
}

func ValidatePassword(password string) (models.Password, error) {
	password = strings.TrimSpace(password)
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	return models.Password(password), nil
}

func SetSalt(salt_str string) {
	salt = []byte(salt_str)
}

// supersecure
func Hashpassword(password models.Password) models.Hash {
	return models.Hash(hex.EncodeToString(argon2.Key([]byte(password), salt, 3, 32*1024, 4, 32)))
}
