package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

func GenerateNumericCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func HashCode(purpose, email, code, secret string) string {
	payload := strings.ToLower(strings.TrimSpace(email)) + "." + purpose + "." + code + "." + secret
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func VerifyCodeHash(purpose, email, code, secret, expected string) bool {
	actual := HashCode(purpose, email, code, secret)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}
