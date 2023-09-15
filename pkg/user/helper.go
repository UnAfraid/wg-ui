package user

import (
	"math/rand"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const (
	cost = 12
)

var (
	lowerCharSet = "abcdedfghijklmnopqrst"
	upperCharSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberSet    = "0123456789"
	allCharSet   = lowerCharSet + upperCharSet + numberSet
)

func generateRandomPassword(passwordLength, minNum, minUpperCase int) string {
	var pb strings.Builder

	// Set numeric
	for i := 0; i < minNum; i++ {
		random := rand.Intn(len(numberSet))
		pb.WriteString(string(numberSet[random]))
	}

	// Set uppercase
	for i := 0; i < minUpperCase; i++ {
		random := rand.Intn(len(upperCharSet))
		pb.WriteString(string(upperCharSet[random]))
	}

	remainingLength := passwordLength - minNum - minUpperCase
	for i := 0; i < remainingLength; i++ {
		random := rand.Intn(len(allCharSet))
		pb.WriteString(string(allCharSet[random]))
	}
	inRune := []rune(pb.String())
	rand.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}

func generatePassword(password []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, cost)
}

func checkPassword(hashedPassword, password []byte) error {
	err := bcrypt.CompareHashAndPassword(hashedPassword, password)
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return ErrInvalidCredentials
	}
	if err != nil {
		logrus.
			WithError(err).
			WithField("hashed_password", hashedPassword).
			WithField("password_len", len(password)).
			Error("failed to compare password")
		return ErrInvalidCredentials
	}
	return nil
}
