package utils

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

func EncodePassword(password string) (string, error) {
	//Create a salt of 16 bytes
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return "", ErrorHandler(err, "Unable to generate Salt for hashing password.")
	}

	//create hash of password and salt. Encode salt and hash
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	encodedSalt := base64.StdEncoding.EncodeToString(salt)
	encodedHash := base64.StdEncoding.EncodeToString(hash)

	return fmt.Sprintf("%s.%s", encodedSalt, encodedHash), nil
}

func VerifyPassword(enteredPassword, hashedPassword string) error {
	parts := strings.Split(hashedPassword, ".")
	if len(parts) != 2 {
		return ErrorHandler(fmt.Errorf("Error in hashed password"), "Error in hashed password")
	}

	encodedSalt := parts[0]
	encodedHash := parts[1]

	decodedSalt, err := base64.StdEncoding.DecodeString(encodedSalt)
	if err != nil {
		return ErrorHandler(err, "Error in decoding the salt")
	}

	decodedHashFromDb, err := base64.StdEncoding.DecodeString(encodedHash)
	if err != nil {
		return ErrorHandler(err, "Error in decoding the hashed password from DB")
	}

	hashFromEnteredPassword := argon2.IDKey([]byte(enteredPassword), decodedSalt, 1, 64*1024, 4, 32)

	if len(hashFromEnteredPassword) != len(decodedHashFromDb) {
		return ErrorHandler(fmt.Errorf("Incorrect password"), "Incorrect Password")
	}

	if subtle.ConstantTimeCompare(hashFromEnteredPassword, decodedHashFromDb) == 1 {
		return nil
	}
	return ErrorHandler(fmt.Errorf("Incorrect password"), "Incorrect Password")
}
