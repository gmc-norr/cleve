package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

const (
	maxPasswordLength = 64
	lowercase         = "abcdefghijklmnopqrstuvwxyz"
	uppercase         = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers           = "0123456789"
	special           = "!@#$%^&*()-_=+[]{}|;:,.<>?"
)

func generatePassword(length int, includeSpecial bool) (string, error) {
	chars := lowercase + uppercase + numbers
	if includeSpecial {
		chars += special
	}

	if length > maxPasswordLength {
		return "", fmt.Errorf("password longer than max length: %d vs %d", length, maxPasswordLength)
	}

	password := make([]byte, length)
	maxIdx := big.NewInt(int64(len(chars) - 1))

	for i := range length {
		charIdx, err := rand.Int(rand.Reader, maxIdx)
		if err != nil {
			return "", err
		}
		password[i] = chars[charIdx.Int64()]
	}

	return string(password), nil
}

func GenerateTempPassword() (string, error) {
	// b := make([]byte, 8)
	// _, _ = rand.Read(b)
	// return fmt.Sprintf("%x", b)
	return generatePassword(8, false)
}

func HashPassword(password []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
}

func VerifyPassword(hashedPassword string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}
